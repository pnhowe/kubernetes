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

	contractorv1 "t3kton.com/api/v1"
	"t3kton.com/pkg/contractor"
)

func extractID(value string) string {
	if value == "" {
		return ""
	}
	return strings.Split(value, ":")[1]
}

// nolint:unused
// log is for logging in this package.
var structurelog = logf.Log.WithName("structure-resource")

// SetupStructureWebhookWithManager registers the webhook for Structure in the manager.
func SetupStructureWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&contractorv1.Structure{}).
		WithValidator(&StructureCustomValidator{}).
		WithDefaulter(&StructureCustomDefaulter{}).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-contractor-t3kton-com-v1-structure,mutating=true,failurePolicy=fail,groups=contractor.t3kton.com,resources=structures,verbs=create,versions=v1,name=vstructure.kb.io,sideEffects=None,admissionReviewVersions=v1

// StructureCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind Structure when those are created
type StructureCustomDefaulter struct {
}

// Default implements webhook.Defaulter so a webhook will be registered for the type
// We will copy the State, BluePrint, and ConfigValues from contractor if they are blank
func (d *StructureCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	fmt.Println("--------------------------------------------------------------------- Defaulter called")
	structure, ok := obj.(*contractorv1.Structure)
	if !ok {
		return fmt.Errorf("expected an Structure object but got %T", obj)
	}

	structurelog.Info("Defaulting for Structure", "name", structure.Name)

	if structure.Spec.ID == 0 {
		return fmt.Errorf("ID not set")
	}

	if structure.Spec.State != "" && structure.Spec.BluePrint != "" && structure.Spec.ConfigValues != nil {
		structurelog.Info("No Defaulting needed")
		return nil
	}

	client := contractor.GetClient(ctx)

	structurelog.Info("Getting Structure")
	upstreamStructure, err := client.BuildingStructureGet(ctx, structure.Spec.ID)
	if err != nil {
		return fmt.Errorf("unable to get structure '%d', err: %s", structure.Spec.ID, err)
	}

	// State, Blueprint, configvalues should come from curent contractor state if they are not set
	if structure.Spec.State == "" {
		structurelog.Info("setting", "state", *upstreamStructure.State)
		structure.Spec.State = *upstreamStructure.State
	}

	if structure.Spec.BluePrint == "" {
		structurelog.Info("setting", "blueprint", extractID(*upstreamStructure.Blueprint))
		structure.Spec.BluePrint = extractID(*upstreamStructure.Blueprint)
	}

	if structure.Spec.ConfigValues == nil {
		structurelog.Info("setting", "config values", *upstreamStructure.ConfigValues)
		structure.Spec.ConfigValues = make(map[string]contractorv1.ConfigValue, len(*upstreamStructure.ConfigValues))
		for key, val := range *upstreamStructure.ConfigValues {
			structure.Spec.ConfigValues[key] = contractorv1.ConfigValueFromContractor(val)
		}
	}

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-contractor-t3kton-com-v1-structure,mutating=false,failurePolicy=fail,sideEffects=None,groups=contractor.t3kton.com,resources=structures,verbs=create;update,versions=v1,name=vstructure-v1.kb.io,admissionReviewVersions=v1

// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type StructureCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &StructureCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Structure.
func (v *StructureCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	structure, ok := obj.(*contractorv1.Structure)
	if !ok {
		return nil, fmt.Errorf("expected a Structure object but got %T", obj)
	}
	structurelog.Info("Validation for Structure upon creation", "name", structure.GetName())

	client := contractor.GetClient(ctx)
	return nil, apierrors.NewAggregate(structure.ValidateStructure(ctx, client))
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Structure.
func (v *StructureCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	newStructure, ok := newObj.(*contractorv1.Structure)
	if !ok {
		return nil, fmt.Errorf("expected a Structure object for the newObj but got %T", newObj)
	}
	structurelog.Info("Validation for Structure upon update", "name", newStructure.GetName())

	oldStructure, ok := oldObj.(*contractorv1.Structure)
	if !ok {
		return nil, fmt.Errorf("expected a Structure object for the oldObj but got %T", oldObj)
	}

	client := contractor.GetClient(ctx)
	return nil, apierrors.NewAggregate(newStructure.ValidateChanges(ctx, client, oldStructure))
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Structure.
func (v *StructureCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	structure, ok := obj.(*contractorv1.Structure)
	if !ok {
		return nil, fmt.Errorf("expected a Structure object but got %T", obj)
	}
	structurelog.Info("Validation for Structure upon deletion", "name", structure.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	// TODO: Do we want to make sure the structure is planned before deleting?

	return nil, nil
}
