/*
Copyright 2024.

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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"
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
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=contractor.t3kton.com,resources=structures,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=contractor.t3kton.com,resources=structures/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=contractor.t3kton.com,resources=structures/finalizers,verbs=update

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile
func (r *StructureReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("structure", req.NamespacedName)
	log.Info("Reconciling Structure", "request", req)

	var structure contractorv1.Structure

	err := r.Get(ctx, req.NamespacedName, &structure)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	fmt.Println("rev 0:", structure.ResourceVersion)
	fmt.Printf("spec: %+v\n", structure.Spec)

	if structure.Spec.ID == 0 {
		log.Info("ID must be specified")
		return ctrl.Result{}, fmt.Errorf("ID Not Specified")
	}

	if (structure.Spec.State == "") || (structure.Spec.BluePrint == "") {
		log.Info("Structure is not fully defined")
		return ctrl.Result{RequeueAfter: time.Second * 60}, nil // wait for the State and BluePrint to be defined
	}

	client := contractor.GetClient(ctx)

	status := contractorv1.StructureStatus{}
	err = updateStatus(ctx, log, client, structure.Spec.ID, &status)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "update status faild")
	}

	diff := cmp.Diff(structure.Status, status)
	if diff != "" {
		fmt.Println(diff)
		log.Info("Status Changed", "diff", diff)
		status.DeepCopyInto(&structure.Status)

		fmt.Println("Update")
		err = r.Status().Update(ctx, &structure)
		fmt.Println("err:", err)
		if apierrors.IsConflict(err) {
			log.Info("Structure Changed on us, will try again")
			return ctrl.Result{Requeue: true}, nil
		}

		if err != nil {
			return ctrl.Result{}, errors.Wrap(err, "update status faild")
		}

		r.publishEvent(ctx, log, &structure, "StatusChanged", "status changed")
		return ctrl.Result{Requeue: true}, nil
	}

	fmt.Println("No diff")
	if structure.Status.Job != nil {
		return ctrl.Result{RequeueAfter: time.Second * 30}, nil
	}

	// Check Config Values, if need changing, change them then requeue, no delay

	// Wait for the job to be cleared up and the state to be set
	if (structure.Status.State == structure.Spec.State) && (structure.Status.BluePrint == structure.Spec.BluePrint) {
		r.publishEvent(ctx, log, &structure, "ReconcileComplete", "reconcile complete")
		log.Info("Reconciled Structure")
		return ctrl.Result{}, nil
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

	jobID, err := r.startJob(ctx, log, client, structure.Spec.ID, jobName)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "create job faild")
	}

	r.publishEvent(ctx, log, &structure, "JobCreated", "job '"+jobName+"' created, ID:"+strconv.Itoa(jobID))
	return ctrl.Result{Requeue: true}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *StructureReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		//WithOptions(controller.Options{MaxConcurrentReconciles: 8}).
		For(&contractorv1.Structure{}).
		Complete(r)
}

// func (r *StructureReconciler) ownObject(ctx context.Context, cr *contractorv1.Structure, obj client.Object) error {

// 	err := ctrl.SetControllerReference(cr, obj, r.Scheme)
// 	if err != nil {
// 		return err
// 	}
// 	return r.Update(ctx, obj)
// }

func (r *StructureReconciler) startJob(ctx context.Context, log logr.Logger, client *cclient.Contractor, ID int, jobName string) (int, error) {
	log.Info("job start", "structure", ID, "name", jobName)
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

func (r *StructureReconciler) publishEvent(ctx context.Context, log logr.Logger, structure *contractorv1.Structure, reason, message string) {
	log.Info("Event", "reason", reason, "message", message)
	t := metav1.Now()

	event := corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: reason + "-",
			Namespace:    structure.Namespace,
		},
		InvolvedObject: corev1.ObjectReference{
			Kind:            structure.Kind,
			Namespace:       structure.Namespace,
			Name:            structure.Name,
			UID:             structure.UID,
			APIVersion:      structure.APIVersion,
			ResourceVersion: structure.ResourceVersion,
		},
		Reason:  reason,
		Message: message,
		Source: corev1.EventSource{
			Component: "t3kton-structure-controller",
		},
		FirstTimestamp:      t,
		LastTimestamp:       t,
		Count:               1,
		Type:                corev1.EventTypeNormal,
		ReportingController: "t3kton.com/structure-controller",
		Related:             structure.Spec.ConsumerRef,
	}

	err := r.Create(ctx, &event)
	if err != nil {
		log.Info("failed to record event, ignoring", "reason", reason, "message", message, "error", err)
	}
}

func updateStatus(ctx context.Context, log logr.Logger, client *cclient.Contractor, ID int, status *contractorv1.StructureStatus) error {
	log.Info("Getting Structure", "id", ID)

	structure, err := client.BuildingStructureGet(ctx, ID)
	if err != nil {
		return err
	}

	log.Info("Getting Foundation", "id", *structure.Foundation)
	foundation, err := client.BuildingFoundationGetURI(ctx, *structure.Foundation)
	if err != nil {
		return err
	}

	updateStructureStatus(structure, status)

	updateFoundationStatus(foundation, status)

	log.Info("Getting Job", "structure", ID)
	jobURI, err := structure.CallGetJob(ctx)
	if err != nil {
		return err
	}
	log.Info("Job Info", "URI", jobURI)

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

	if len(*structure.ConfigValues) > 0 {
		status.ConfigValues = make(map[string]contractorv1.ConfigValue, len(*structure.ConfigValues))
		for key, val := range *structure.ConfigValues {
			status.ConfigValues[key] = contractorv1.FromInterface(val)
		}
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

/// also need events
