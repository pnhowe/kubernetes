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

package v1

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	apierrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"t3kton.com/pkg/contractor"
)

func extractID(value string) string {
	if value == "" {
		return ""
	}
	return strings.Split(value, ":")[1]
}

// log is for logging in this package.
var structurelog = logf.Log.WithName("structure-resource")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *Structure) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-contractor-t3kton-com-v1-structure,mutating=true,failurePolicy=fail,groups=contractor.t3kton.com,resources=structures,verbs=create,versions=v1,name=vstructure.kb.io,sideEffects=None,admissionReviewVersions=v1

var _ webhook.Defaulter = &Structure{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Structure) Default() {
	structurelog.Info("default", "name", r.Name)

	ctx := context.TODO()
	client := contractor.GetClient(ctx)

	structurelog.Info("Getting Structure")
	structure, err := client.BuildingStructureGet(ctx, r.Spec.ID)
	if err != nil { // Hopfully the validation logic will catch the fact this dosen't exist
		return
	}

	// State, Blueprint, configvalues should come from curent contractor state if they are not set
	if r.Spec.State == "" {
		structurelog.Info("setting", "state", *structure.State)
		r.Spec.State = *structure.State
	}

	if r.Spec.BluePrint == "" {
		structurelog.Info("setting", "blueprint", extractID(*structure.Blueprint))
		r.Spec.BluePrint = extractID(*structure.Blueprint)
	}

	if r.Spec.ConfigValues == nil {
		structurelog.Info("setting", "config values", *structure.ConfigValues)
		r.Spec.ConfigValues = make(map[string]contractor.ConfigValue, len(*structure.ConfigValues))
		for key, val := range *structure.ConfigValues {
			r.Spec.ConfigValues[key] = contractor.FromInterface(val)
		}
	}
}

//+kubebuilder:webhook:path=/validate-contractor-t3kton-com-v1-structure,mutating=false,failurePolicy=fail,sideEffects=None,groups=contractor.t3kton.com,resources=structures,verbs=create;update;delete,versions=v1,name=vstructure.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Structure{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Structure) ValidateCreate() (admission.Warnings, error) {
	structurelog.Info("validate create", "name", r.Name)

	ctx := context.TODO()
	client := contractor.GetClient(ctx)

	return nil, apierrors.NewAggregate(r.validateStructure(ctx, client))
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Structure) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	structurelog.Info("validate update", "name", r.Name)

	ctx := context.TODO()
	client := contractor.GetClient(ctx)

	structure, casted := old.(*Structure)
	if !casted {
		structurelog.Error(fmt.Errorf("old object conversion error for %s/%d", r.Namespace, r.Spec.ID), "validate update error")
		return nil, nil
	}
	return nil, apierrors.NewAggregate(r.validateChanges(ctx, client, structure))
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Structure) ValidateDelete() (admission.Warnings, error) {
	structurelog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}
