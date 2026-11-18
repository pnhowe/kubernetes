/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"t3kton.com/pkg/contractor"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	cclient "github.com/t3kton/contractor_goclient"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	contractorv1 "t3kton.com/api/v1"

	"github.com/go-logr/logr"
)

// requeueSoon is the delay before reconciling again after a step that needs the next step to be done, instead of waiting for the resync.
// It gives the cache time to see our own changes and keeps from hammering contractor.
const requeueSoon = 500 * time.Millisecond

// SyncMode controls which side is the source of truth for a Structure
type SyncMode string

const (
	// SyncModeEnforce makes the Structure resource the source of truth, contractor is changed to match the spec.
	SyncModeEnforce SyncMode = "enforce"
	// SyncModeFollow makes contractor the source of truth, the spec is changed to match contractor.
	// In this mode the controller never changes anything in contractor.
	SyncModeFollow SyncMode = "follow"
)

// ParseSyncMode validates a sync mode string, the empty string is SyncModeEnforce
func ParseSyncMode(value string) (SyncMode, error) {
	switch SyncMode(value) {
	case "", SyncModeEnforce:
		return SyncModeEnforce, nil
	case SyncModeFollow:
		return SyncModeFollow, nil
	}
	return "", fmt.Errorf("invalid sync mode '%s', must be '%s' or '%s'", value, SyncModeEnforce, SyncModeFollow)
}

// StructureReconciler reconciles a Structure object
type StructureReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder events.EventRecorder
	// SyncMode defaults to SyncModeEnforce
	SyncMode SyncMode
	// ResyncInterval is how often an idle Structure is checked against contractor again, 0 disables the periodic resync.
	// Without it changes made directly in contractor are not noticed until the Structure resource is touched.
	ResyncInterval time.Duration
}

// +kubebuilder:rbac:groups=contractor.t3kton.com,resources=structures,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=contractor.t3kton.com,resources=structures/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=events.k8s.io,resources=events,verbs=create;patch

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.2/pkg/reconcile
func (r *StructureReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling Structure", "request", req)

	var structure contractorv1.Structure

	err := r.Get(ctx, req.NamespacedName, &structure)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Deleting a Structure only removes it from k8s, it is never destroyed in contractor, so there is nothing to do
	if !structure.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	// This should never happen, but just incase
	if structure.Spec.ID == 0 {
		logger.Info("ID must be specified")
		return ctrl.Result{}, fmt.Errorf("ID Not Specified")
	}

	if (structure.Spec.State == "") || (structure.Spec.BluePrint == "") {
		logger.Info("Structure is not fully defined")
		// the defaulting webhook fills these in, so this only happens if the webhook is bypassed.
		// Changing the spec triggers a new reconcile, so there is no need to requeue, the error is just so it is noticed
		return ctrl.Result{}, fmt.Errorf("structure is not fully defined")
	}

	client, err := contractor.GetClient(ctx)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "unable to get contractor client")
	}

	logger.Info("Getting Structure", "id", structure.Spec.ID)
	t3kton_structure, err := client.BuildingStructureGet(ctx, structure.Spec.ID)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "get structure faild")
	}

	status := contractorv1.StructureStatus{}
	err = updateStatus(ctx, logger, client, t3kton_structure, &status)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "update status faild")
	}

	// See if an existing job has finished
	if structure.Status.Job != nil && status.Job == nil {
		r.Recorder.Eventf(&structure, nil, corev1.EventTypeNormal, "JobFinished", "Reconcile", "Job '%s' finished", structure.Status.Job.Script)

		structure.Status.Job = nil
		err = r.Status().Update(ctx, &structure)
		if apierrors.IsConflict(err) {
			logger.Info("Structure Changed on us")
		}
		if err != nil {
			logger.Error(err, "updating job status failed")
		}

		return ctrl.Result{RequeueAfter: requeueSoon}, nil
	}

	// see if the state of the structure/foundation/job on contractor	is different from what we have
	// the status is our internal copy of the existing status of the structure
	// we could break this up to compare ConfigValues, state, job, etc sepertaly
	changed := syncStatus(&structure, &status)
	dirty := len(changed) > 0

	if dirty {
		logger.Info("Status Change Detected", "changed", changed)
		err = r.Status().Update(ctx, &structure)
		if apierrors.IsConflict(err) {
			logger.Info("Structure Changed on us, will try again")
			return ctrl.Result{RequeueAfter: requeueSoon}, nil
		}

		if err != nil {
			return ctrl.Result{}, errors.Wrap(err, "update status faild")
		}
		return ctrl.Result{RequeueAfter: requeueSoon}, nil
	}

	// if there is a job, requeue and wait for the job to finish before we do anything else
	if structure.Status.Job != nil {
		return ctrl.Result{RequeueAfter: time.Second * 30}, nil // contractor has no way to tell us the job finished, so poll
	}

	if r.SyncMode == SyncModeFollow {
		return r.followContractor(ctx, logger, &structure, &status)
	}

	return r.enforceSpec(ctx, logger, client, &structure, &status, t3kton_structure)
}

// enforceSpec is the SyncModeEnforce reconcile, status has already been updated from contractor, and there is no job.
// Contractor is changed to match the spec, one step at a time, each step requeues so the status is refreshed.
func (r *StructureReconciler) enforceSpec(ctx context.Context, logger logr.Logger, client *cclient.Contractor,
	structure *contractorv1.Structure, status *contractorv1.StructureStatus, t3kton_structure *cclient.BuildingStructure) (ctrl.Result, error) {
	// Check Config Values, if need changing, change them then requeue, no delay
	// This is the only thing in the spec that does not require a job
	if !cmp.Equal(structure.Spec.ConfigValues, status.ConfigValues) {
		// We only want to update the config values, make an empty copy with only config values so only thoes get updated
		tmp_structure := client.BuildingStructureNewWithID(*t3kton_structure.ID)
		tmp_ConfigValues := structure.Spec.ConfigValues.ToContractor()
		tmp_structure.ConfigValues = &tmp_ConfigValues
		err := tmp_structure.Update(ctx)
		if err != nil {
			return ctrl.Result{}, errors.Wrap(err, "update config values on contractor faild")
		}
		logger.Info("ConfigValues updated")
		return ctrl.Result{RequeueAfter: requeueSoon}, nil
	}

	if (structure.Status.State == structure.Spec.State) && (structure.Status.BluePrint == structure.Spec.BluePrint) {
		logger.Info("Reconciled Structure")
		return r.idleResult(), nil
	}

	blueprintChanged := structure.Status.BluePrint != structure.Spec.BluePrint

	// If we are in planned state, we can update the blueprint, no destroy job needed
	if blueprintChanged && structure.Status.State == contractorv1.StatePlanned {
		tmp_structure := client.BuildingStructureNewWithID(*t3kton_structure.ID)
		tmp_blueprint := "/api/v1/BluePrint/StructureBluePrint:" + structure.Spec.BluePrint + ":"
		tmp_structure.Blueprint = &tmp_blueprint
		err := tmp_structure.Update(ctx)
		if err != nil {
			return ctrl.Result{}, errors.Wrap(err, "update blueprint on contractor faild")
		}
		logger.Info("BluePrint updated")
		return ctrl.Result{RequeueAfter: requeueSoon}, nil
	}

	// Guess we need to make a job then.
	// A built structure has to be destroyed to get to planned, which is also what is needed to change the blueprint,
	// once it is planned the blueprint is changed (above) and then a create job is made to get back to built.
	var jobName string
	if structure.Status.State == contractorv1.StateBuilt && (structure.Spec.State == contractorv1.StatePlanned || blueprintChanged) {
		jobName = "destroy"
	} else if structure.Status.State == contractorv1.StatePlanned && structure.Spec.State == contractorv1.StateBuilt {
		jobName = "create"
	} else {
		return ctrl.Result{}, fmt.Errorf("invalid target state, current state '%s', target state '%s'", structure.Status.State, structure.Spec.State)
	}

	jobID, err := r.startJob(ctx, logger, client, structure.Spec.ID, jobName)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "job create faild")
	}
	r.Recorder.Eventf(structure, nil, corev1.EventTypeNormal, "JobCreated", "Reconcile", "job '%s' created, ID:%d", jobName, jobID)
	return ctrl.Result{RequeueAfter: requeueSoon}, nil
}

// idleResult is the result to use when there is nothing left to do, schedules the next resync if enabled
func (r *StructureReconciler) idleResult() ctrl.Result {
	return ctrl.Result{RequeueAfter: r.ResyncInterval}
}

// followContractor is the SyncModeFollow reconcile, status has already been updated from contractor, and there is no job.
// The spec is changed to match contractor, nothing is changed in contractor.
// The state and blueprint are changed in separate updates, so each update passes the validation webhook
// the same way a person making the changes one at a time would.
func (r *StructureReconciler) followContractor(ctx context.Context, logger logr.Logger, structure *contractorv1.Structure, status *contractorv1.StructureStatus) (ctrl.Result, error) {
	changed := []string{}

	if structure.Spec.State != status.State {
		structure.Spec.State = status.State
		changed = append(changed, "State")
	} else if structure.Spec.BluePrint != status.BluePrint {
		structure.Spec.BluePrint = status.BluePrint
		changed = append(changed, "BluePrint")
	}

	if !cmp.Equal(structure.Spec.ConfigValues, status.ConfigValues) {
		structure.Spec.ConfigValues = status.ConfigValues.DeepCopy()
		changed = append(changed, "ConfigValues")
	}

	if len(changed) == 0 {
		logger.Info("Structure matches contractor")
		return r.idleResult(), nil
	}

	logger.Info("Updating spec to match contractor", "changed", changed)
	err := r.Update(ctx, structure)
	if apierrors.IsConflict(err) {
		logger.Info("Structure Changed on us, will try again")
		return ctrl.Result{RequeueAfter: requeueSoon}, nil
	}
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "update spec to match contractor faild")
	}

	r.Recorder.Eventf(structure, nil, corev1.EventTypeNormal, "SpecSynced", "Reconcile", "spec updated to match contractor: %s", strings.Join(changed, ", "))
	return ctrl.Result{RequeueAfter: requeueSoon}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *StructureReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}). // contractor is the shared resource, one at a time keeps the load down
		For(&contractorv1.Structure{}).
		Named("structure").
		Complete(r)
}

// syncStatus copies what contractor reported (status) into the Structure's status, returning what changed
func syncStatus(structure *contractorv1.Structure, status *contractorv1.StructureStatus) []string {
	changed := []string{}
	if structure.Status.State != status.State {
		structure.Status.State = status.State
		changed = append(changed, "State")
	}
	if structure.Status.BluePrint != status.BluePrint {
		structure.Status.BluePrint = status.BluePrint
		changed = append(changed, "BluePrint")
	}
	if structure.Status.Hostname != status.Hostname {
		structure.Status.Hostname = status.Hostname
		changed = append(changed, "Hostname")
	}
	if !cmp.Equal(status.ConfigValues, structure.Status.ConfigValues) {
		structure.Status.ConfigValues = status.ConfigValues.DeepCopy()
		changed = append(changed, "ConfigValues")
	}
	// This one is just so we can watch the job come and go, the job going away is handled in Reconcile
	// for somereason cmp.Equal(nil, nil) is false here
	if !cmp.Equal(structure.Status.Job, status.Job) && status.Job != nil {
		structure.Status.Job = status.Job.DeepCopy()
		changed = append(changed, "Job")
	}
	// These two can't be changed in k8s, we are replicating them here for information purposes
	if structure.Status.Foundation != status.Foundation {
		structure.Status.Foundation = status.Foundation
		changed = append(changed, "Foundation")
	}
	if structure.Status.FoundationBluePrint != status.FoundationBluePrint {
		structure.Status.FoundationBluePrint = status.FoundationBluePrint
		changed = append(changed, "FoundationBluePrint")
	}
	return changed
}

func (r *StructureReconciler) startJob(ctx context.Context, logger logr.Logger, client *cclient.Contractor, ID int, jobName string) (int, error) {
	logger.Info("job start", "structure", ID, "name", jobName)
	structure := client.BuildingStructureNewWithID(ID)

	var err error
	var jobID int
	switch jobName {
	case "create":
		jobID, err = structure.CallDoCreate(ctx)
	case "destroy":
		jobID, err = structure.CallDoDestroy(ctx)
	default:
		return 0, fmt.Errorf("invalid job name '%s'", jobName)
	}
	if err != nil {
		return 0, errors.Wrap(err, "do job failed")
	}

	return jobID, nil
}

func updateStatus(ctx context.Context, logger logr.Logger, client *cclient.Contractor, structure *cclient.BuildingStructure, status *contractorv1.StructureStatus) error {

	logger.Info("Getting Foundation", "id", *structure.Foundation)
	foundation, err := client.BuildingFoundationGetURI(ctx, *structure.Foundation)
	if err != nil {
		return err
	}

	updateStructureStatus(structure, status)

	updateFoundationStatus(foundation, status)

	logger.Info("Getting Job", "structure", structure.ID)
	jobURI, err := structure.CallGetJob(ctx)
	if err != nil {
		return err
	}

	if jobURI == "" {
		status.Job = nil
		return nil
	}

	job, err := client.ForemanStructureJobGetURI(ctx, jobURI)
	if err != nil {
		return err
	}

	updateJobStatus(job, status)

	return nil
}

func updateStructureStatus(structure *cclient.BuildingStructure, status *contractorv1.StructureStatus) {
	status.State = *structure.State
	status.Hostname = *structure.Hostname
	status.BluePrint = strings.Split(*structure.Blueprint, ":")[1]
	status.Foundation = *structure.Foundation
	if structure.ConfigValues != nil {
		status.ConfigValues = contractorv1.ConfigValuesFromContractor(*structure.ConfigValues)
	} else {
		status.ConfigValues = contractorv1.ConfigValues{}
	}
}

func updateFoundationStatus(foundation *cclient.BuildingFoundation, status *contractorv1.StructureStatus) {
	status.Foundation = *foundation.Locator
	status.FoundationBluePrint = strings.Split(*foundation.Blueprint, ":")[1]
}

func updateJobStatus(job *cclient.ForemanStructureJob, status *contractorv1.StructureStatus) {
	if status.Job == nil {
		status.Job = &contractorv1.JobStatus{}
	}

	status.Job.State = *job.State
	status.Job.Script = *job.ScriptName
	status.Job.Message = *job.Message
	status.Job.CanStart = *job.CanStart
	status.Job.Created = job.Created.Format(time.RFC3339)
	status.Job.LastUpdated = job.Updated.Format(time.RFC3339)

	// Status is a list of scope entries, the first is the outermost scope and has the overall progress
	// each entry is like: {'percent': 12.5, 'operation': 'Scope', 'parameters': {'time_elapsed': '00:02', 'time_remaining': '00:07'}}
	status.Job.Progress = "0"
	status.Job.MaxTimeRemaining = ""

	if job.Status != nil && len(*job.Status) > 0 {
		top := (*job.Status)[0]
		if percent, ok := top["percent"].(float64); ok {
			status.Job.Progress = strconv.FormatFloat(percent, 'f', 1, 64)
		}
		if params, ok := top["parameters"].(map[string]any); ok {
			if remaining, ok := params["time_remaining"].(string); ok {
				status.Job.MaxTimeRemaining = remaining
			}
		}
	}

	if status.Job.MaxTimeRemaining == "" && status.Job.Progress == "100.0" {
		status.Job.MaxTimeRemaining = "00:00"
	}
}
