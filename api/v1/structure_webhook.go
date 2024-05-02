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
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var structurelog = logf.Log.WithName("structure-resource")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *Structure) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
//+kubebuilder:webhook:path=/validate-contractor-t3kton-com-v1-structure,mutating=false,failurePolicy=fail,sideEffects=None,groups=contractor.t3kton.com,resources=structures,verbs=create;update;delete,versions=v1,name=vstructure.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Structure{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Structure) ValidateCreate() (admission.Warnings, error) {
	structurelog.Info("validate create", "name", r.Name)

	return nil, kerrors.NewAggregate(r.validateStructure())

}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Structure) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	structurelog.Info("validate update", "name", r.Name)

	structure, casted := old.(*Structure)
	if !casted {
		structurelog.Error(fmt.Errorf("old object conversion error for %s/%d", r.Namespace, r.Spec.ID), "validate update error")
		return nil, nil
	}
	return nil, kerrors.NewAggregate(r.validateChanges(structure))
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Structure) ValidateDelete() (admission.Warnings, error) {
	structurelog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}
