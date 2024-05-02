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
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var structurelog = log.Log.WithName("webhooks").WithName("Structure")

//+kubebuilder:webhook:verbs=create;update,path=/validate-contractor-t3kton-com-v1,mutating=false,failurePolicy=fail,sideEffects=none,admissionReviewVersions=v1;v1,groups=contractor.t3kton.com,resources=structure,versions=v1,name=structure.contractor.t3kton.com

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (s *Structure) ValidateCreate() (admission.Warnings, error) {
	structurelog.Info("validate create", "namespace", s.Namespace, "ID", s.Spec.ID)
	return nil, kerrors.NewAggregate(s.validateStructure())
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (s *Structure) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	structurelog.Info("validate update", "namespace", s.Namespace, "ID", s.Spec.ID)
	structure, casted := old.(*Structure)
	if !casted {
		structurelog.Error(fmt.Errorf("old object conversion error for %s/%d", s.Namespace, s.Spec.ID), "validate update error")
		return nil, nil
	}
	return nil, kerrors.NewAggregate(s.validateChanges(structure))
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (s *Structure) ValidateDelete() (admission.Warnings, error) {
	return nil, nil
}
