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

	apierrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
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
	return ctrl.NewWebhookManagedBy(mgr, &contractorv1.Structure{}).
		WithValidator(&StructureCustomValidator{}).
		WithDefaulter(&StructureCustomDefaulter{}).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-contractor-t3kton-com-v1-structure,mutating=true,failurePolicy=fail,sideEffects=None,groups=contractor.t3kton.com,resources=structures,verbs=create,versions=v1,name=mstructure-v1.kb.io,admissionReviewVersions=v1

// StructureCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind Structure when those are created
type StructureCustomDefaulter struct {
}

var _ admission.Defaulter[*contractorv1.Structure] = &StructureCustomDefaulter{}

// Default implements admission.Defaulter so a webhook will be registered for the Kind Structure.
// We will copy the State, BluePrint, and ConfigValues from contractor if they are blank
func (d *StructureCustomDefaulter) Default(ctx context.Context, structure *contractorv1.Structure) error {
	structurelog.Info("Defaulting for Structure", "name", structure.GetName())

	if structure.Spec.ID == 0 {
		return fmt.Errorf("ID not set")
	}

	if structure.Spec.State != "" && structure.Spec.BluePrint != "" && structure.Spec.ConfigValues != nil {
		structurelog.Info("No Defaulting needed")
		return nil
	}

	client, err := contractor.GetClient(ctx)
	if err != nil {
		return err
	}

	structurelog.Info("Getting Structure")
	upstreamStructure, err := client.BuildingStructureGet(ctx, structure.Spec.ID)
	if err != nil {
		return fmt.Errorf("unable to get structure '%d', err: %s", structure.Spec.ID, err)
	}

	// State, Blueprint, configvalues should come from current contractor state if they are not set
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

// +kubebuilder:webhook:path=/validate-contractor-t3kton-com-v1-structure,mutating=false,failurePolicy=fail,sideEffects=None,groups=contractor.t3kton.com,resources=structures,verbs=create;update;delete,versions=v1,name=vstructure-v1.kb.io,admissionReviewVersions=v1

// StructureCustomValidator struct is responsible for validating the Structure resource
// when it is created, updated, or deleted.
type StructureCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ admission.Validator[*contractorv1.Structure] = &StructureCustomValidator{}

// ValidateCreate implements admission.Validator so a webhook will be registered for the type Structure.
func (v *StructureCustomValidator) ValidateCreate(ctx context.Context, structure *contractorv1.Structure) (admission.Warnings, error) {
	structurelog.Info("Validation for Structure upon creation", "name", structure.GetName())

	client, err := contractor.GetClient(ctx)
	if err != nil {
		return nil, err
	}
	return nil, apierrors.NewAggregate(structure.ValidateStructure(ctx, client))
}

// ValidateUpdate implements admission.Validator so a webhook will be registered for the type Structure.
func (v *StructureCustomValidator) ValidateUpdate(ctx context.Context, oldStructure, newStructure *contractorv1.Structure) (admission.Warnings, error) {
	structurelog.Info("Validation for Structure upon update", "name", newStructure.GetName())

	client, err := contractor.GetClient(ctx)
	if err != nil {
		return nil, err
	}
	return nil, apierrors.NewAggregate(newStructure.ValidateChanges(ctx, client, oldStructure))
}

// ValidateDelete implements admission.Validator so a webhook will be registered for the type Structure.
func (v *StructureCustomValidator) ValidateDelete(ctx context.Context, structure *contractorv1.Structure) (admission.Warnings, error) {
	structurelog.Info("Validation for Structure upon deletion", "name", structure.GetName())

	return nil, apierrors.NewAggregate(structure.CanDelete(ctx))
}
