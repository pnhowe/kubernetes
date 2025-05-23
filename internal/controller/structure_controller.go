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
	"regexp"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
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

// StructureReconciler reconciles a Structure object
type StructureReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=contractor.t3kton.com,resources=structures,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=contractor.t3kton.com,resources=structures/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=contractor.t3kton.com,resources=structures/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

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

	// This should never happen, but just incase
	if structure.Spec.ID == 0 {
		logger.Info("ID must be specified")
		return ctrl.Result{}, fmt.Errorf("ID Not Specified")
	}

	if (structure.Spec.State == "") || (structure.Spec.BluePrint == "") {
		logger.Info("Structure is not fully defined")
		//return ctrl.Result{Requeue: true}, nil // wait for the State and BluePrint to be defined, TODO: do we need to requeue here? will this enitiy get auto-requeued when the spec is updated?
		return ctrl.Result{}, fmt.Errorf("structure is not fully defined")
	}

	client := contractor.GetClient(ctx)

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
		r.Recorder.Event(&structure, "Normal", "JobFinished", "Job '"+structure.Status.Job.Script+"' finished")

		structure.Status.Job = nil
		err = r.Status().Update(ctx, &structure)
		if apierrors.IsConflict(err) {
			logger.Info("Structure Changed on us")
		}
		if err != nil {
			logger.Error(err, "updating job status failed")
		}

		return ctrl.Result{Requeue: true}, nil
	}

	// see if the state of the structure/foundation/job on contractor	is different from what we have
	// the status is our internal copy of the existing status of the structure
	// we could break this up to compare ConfigValues, state, job, etc sepertaly
	// TODO: for testing, make sure all these comparision work, expecially the configvalue and job, when empty, nil, maps, slices, maps with maps and slices and nils, "7" != 7 != 7.0, oh my
	dirty := false
	changed := []string{}
	if structure.Status.State != status.State {
		structure.Status.State = status.State
		changed = append(changed, "State")
		dirty = true
	}
	if structure.Status.BluePrint != status.BluePrint {
		structure.Status.BluePrint = status.BluePrint
		changed = append(changed, "BluePrint")
		dirty = true
	}
	if structure.Status.Hostname != status.Hostname {
		structure.Status.Hostname = status.Hostname
		changed = append(changed, "Hostname")
		dirty = true
	}
	if !cmp.Equal(status.ConfigValues, structure.Status.ConfigValues) {
		structure.Status.ConfigValues = status.ConfigValues.DeepCopy()
		changed = append(changed, "ConfigValues")
		dirty = true
	}
	// This one is just so we can watch the job come and go
	// for somereason cmp.Equal(nil, nil) is false here
	if !cmp.Equal(structure.Status.Job, status.Job) && status.Job != nil {
		structure.Status.Job = status.Job.DeepCopy()
		changed = append(changed, "Job")
		dirty = true
	}
	// These two can't be changed in k8s, we are replicating them here for information purposes
	if structure.Status.Foundation != status.Foundation {
		structure.Status.Foundation = status.Foundation
		changed = append(changed, "Foundation")
		dirty = true
	}
	if structure.Status.FoundationBluePrint != status.FoundationBluePrint {
		structure.Status.FoundationBluePrint = status.FoundationBluePrint
		changed = append(changed, "FoundationBluePrint")
		dirty = true
	}

	if dirty {
		logger.Info("Status Change Detected", "changed", changed)
		err = r.Status().Update(ctx, &structure)
		if apierrors.IsConflict(err) {
			logger.Info("Structure Changed on us, will try again")
			return ctrl.Result{Requeue: true}, nil
		}

		if err != nil {
			return ctrl.Result{}, errors.Wrap(err, "update status faild")
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// if there is a job, requeue and wait for the job to finish before we do anything else
	if structure.Status.Job != nil {
		return ctrl.Result{RequeueAfter: time.Second * 30}, nil // TODO: should this be a regular requeue?
	}

	// Check Config Values, if need changing, change them then requeue, no delay
	// This is the only thing in the spec that does not require a job
	if !cmp.Equal(structure.Spec.ConfigValues, status.ConfigValues) {
		// We only want to update the config values, make an empty copy with only config values so only thoes get updated
		tmp_structure := client.BuildingStructureNewWithID(*t3kton_structure.ID)
		tmp_ConfigValues := structure.Spec.ConfigValues.ToContractor()
		tmp_structure.ConfigValues = &tmp_ConfigValues
		_, err := tmp_structure.Update(ctx)
		if err != nil {
			return ctrl.Result{Requeue: false}, errors.Wrap(err, "update config values on contractor faild") // TODO: Check to see if it is something that could be retried
		}
		logger.Info("ConfigValues updated")
		return ctrl.Result{Requeue: true}, nil
	}

	// Wait for the job to be cleared up and the state to be set
	if (structure.Status.State == structure.Spec.State) && (structure.Status.BluePrint == structure.Spec.BluePrint) {
		r.Recorder.Event(&structure, "Normal", "ReconcileComplete", "reconcile complete")
		logger.Info("Reconciled Structure")
		return ctrl.Result{}, nil
	}

	if structure.Status.BluePrint != structure.Spec.BluePrint {
		// If we are allready in planned state, we can update the blueprint, no destroy job needed
		if structure.Status.State == "planned" {
			tmp_structure := client.BuildingStructureNewWithID(*t3kton_structure.ID)
			tmp_blueprint := "/api/v1/BluePrint/StructureBluePrint:" + structure.Spec.BluePrint + ":"
			tmp_structure.Blueprint = &tmp_blueprint
			_, err := tmp_structure.Update(ctx)
			if err != nil {
				return ctrl.Result{Requeue: false}, errors.Wrap(err, "update blueprint on contractor faild") // TODO: Check to see if it is something that could be retried
			}
			logger.Info("BluePrint updated")
			return ctrl.Result{Requeue: true}, nil
		}
		// fallthrough to the destroy job creation
	}

	// Guess we need to make a job then
	var jobName string
	if structure.Spec.State == "built" {
		jobName = "create"
	} else if structure.Spec.State == "planned" {
		jobName = "destroy"
	} else {
		return ctrl.Result{}, fmt.Errorf("invalid target state")
	}

	jobID, err := r.startJob(ctx, logger, client, structure.Spec.ID, jobName)
	if err != nil {
		return ctrl.Result{Requeue: false}, errors.Wrap(err, "job create faild") // TODO: Check to see if it is something that could be retried
	}
	r.Recorder.Event(&structure, "Normal", "JobCreated", "job '"+jobName+"' created, ID:"+strconv.Itoa(jobID))
	return ctrl.Result{Requeue: true}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *StructureReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}). // TODO: rate limiter, make sure it isn't reconciling the same structure multiple times at the same time
		For(&contractorv1.Structure{}).
		Named("structure").
		Complete(r)
}

// func (r *StructureReconciler) ownObject(ctx context.Context, cr *contractorv1.Structure, obj client.Object) error {

// 	err := ctrl.SetControllerReference(cr, obj, r.Scheme)
// 	if err != nil {
// 		return err
// 	}
// 	return r.Update(ctx, obj)
// }

func (r *StructureReconciler) startJob(ctx context.Context, logger logr.Logger, client *cclient.Contractor, ID int, jobName string) (int, error) {
	logger.Info("job start", "structure", ID, "name", jobName)
	structure := client.BuildingStructureNewWithID(ID)

	var err error
	var jobID int
	if jobName == "create" {
		jobID, err = structure.CallDoCreate(ctx)
	} else if jobName == "destroy" {
		jobID, err = structure.CallDoDestroy(ctx)
	} else {
		return 0, fmt.Errorf("invalid job name '" + jobName + "'")
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
	status.ConfigValues = contractorv1.ConfigValuesFromContractor(*structure.ConfigValues)
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

	r, _ := regexp.Compile(`\[\[([0-9\.]+)`)

	jobStatus := r.FindString(*job.Status)
	if jobStatus != "" {
		status.Job.Progress = jobStatus[2:] // skip the leading [[
	} else {
		status.Job.Progress = "0"
	}

	r, _ = regexp.Compile(`'time_remaining': '[0-9:]{5}'`)
	jobStatus = r.FindString(*job.Status)
	if jobStatus != "" {
		status.Job.MaxTimeRemaining = jobStatus[19:24]
	} else if status.Job.Progress == "100.0" {
		status.Job.MaxTimeRemaining = "00:00"
	} else {
		status.Job.MaxTimeRemaining = ""
	}
}

// TODO: also need events
