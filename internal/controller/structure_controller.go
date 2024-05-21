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
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"t3kton.com/pkg/contractor"

	cclient "github.com/t3kton/contractor_goclient"
	contractorv1 "t3kton.com/api/v1"

	"github.com/go-logr/logr"
)

// StructureReconciler reconciles a Structure object
type StructureReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=contractor.t3kton.com,resources=structures,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=contractor.t3kton.com,resources=structures/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=contractor.t3kton.com,resources=structures/finalizers,verbs=update

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile
func (r *StructureReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Reconciling Structure")
	fmt.Println("A")
	var structure contractorv1.Structure
	fmt.Println("B")
	err := r.Get(ctx, req.NamespacedName, &structure)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	fmt.Println("C")
	if (structure.Spec.State == "") || (structure.Spec.BluePrint == "") {
		log.Info("Structure is not fully defined")
		return ctrl.Result{RequeueAfter: time.Second * 30}, nil // wait for the State and BluePrint to be defined
	}
	fmt.Println("D")
	client := contractor.GetClient(ctx)
	fmt.Println("E")
	cStructure, err := updateStructureStatus(ctx, log, client, &structure)
	if err != nil {
		return ctrl.Result{}, err
	}
	fmt.Println("F")
	err = updateJobStatus(ctx, log, client, cStructure, &structure)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.Status().Update(ctx, &structure)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Wait for the job to be cleared up and the state to be set
	if (structure.Status.Job == nil) && (structure.Status.State == structure.Spec.State) && (structure.Status.BluePrint == structure.Spec.BluePrint) {
		log.Info("Reconciled Structure")
		return ctrl.Result{}, nil
	}

	if structure.Status.Job == nil {
		log.Info("Starting Job")
		var err error
		if structure.Spec.State == "built" {
			_, err = cStructure.CallDoCreate(ctx)
		} else if structure.Spec.State == "planned" {
			_, err = cStructure.CallDoDestroy(ctx)
		} else {
			return ctrl.Result{}, fmt.Errorf("invalid target state")
		}

		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{RequeueAfter: time.Second * 30}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *StructureReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: 8}).
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

func updateStructureStatus(ctx context.Context, log logr.Logger, client *cclient.Contractor, structure *contractorv1.Structure) (*cclient.BuildingStructure, error) {
	log.Info("Getting Structure")
	cStructure, err := client.BuildingStructureGet(ctx, structure.Spec.ID)
	if err != nil {
		return nil, err
	}

	structure.Status.State = *cStructure.State
	structure.Status.Hostname = *cStructure.Hostname
	structure.Status.BluePrint = strings.Split(*cStructure.Blueprint, ":")[1]
	structure.Status.Foundation = *cStructure.Foundation

	structure.Status.ConfigValues = make(map[string]contractor.ConfigValue, len(*cStructure.ConfigValues))
	for key, val := range *cStructure.ConfigValues {
		structure.Status.ConfigValues[key] = contractor.FromInterface(val)
	}

	log.Info("Getting Foundation")
	cFoundation, err := client.BuildingFoundationGetURI(ctx, structure.Status.Foundation)
	if err != nil {
		return nil, err
	}

	structure.Status.Foundation = *cFoundation.Locator
	structure.Status.FoundationBluePrint = strings.Split(*cFoundation.Blueprint, ":")[1]

	return cStructure, nil
}

func updateJobStatus(ctx context.Context, log logr.Logger, client *cclient.Contractor, cStructure *cclient.BuildingStructure, structure *contractorv1.Structure) error {
	log.Info("Getting Structure Job")
	jobURI, err := cStructure.CallGetJob(ctx)
	if err != nil {
		return err
	}

	log.Info("Job Info", "URI", jobURI)

	if jobURI == "" {
		structure.Status.Job = nil
		return nil
	} else if structure.Status.Job == nil {
		structure.Status.Job = &contractorv1.JobStatus{}
	}

	job, err := client.ForemanStructureJobGetURI(ctx, jobURI)
	if err != nil {
		return err
	}

	log.Info("Job Info", "name", *job.ScriptName)
	log.Info("Job Info", "state", *job.State)

	structure.Status.Job.State = *job.State
	structure.Status.Job.Script = *job.ScriptName
	structure.Status.Job.Message = *job.Message
	structure.Status.Job.CanStart = *job.CanStart
	structure.Status.Job.Updated = job.Updated.String()

	r, _ := regexp.Compile(`\[\[([0-9\.]+)`)

	status := r.FindString(*job.Status)
	if status != "" {
		structure.Status.Job.Progress = status[2:] // skip the leading [[
	} else {
		structure.Status.Job.Progress = "0"
	}

	r, _ = regexp.Compile(`'time_remaining': '[0-9:]{5}'`)
	status = r.FindString(*job.Status)
	if status != "" {
		structure.Status.Job.MaxTimeRemaining = status[19:24]
	} else {
		structure.Status.Job.MaxTimeRemaining = "<unknwon>"
	}

	return nil
}

/// also need events
