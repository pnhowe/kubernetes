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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// StructureSpec defines the desired state of Structure
type StructureSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	ID int `json:"id,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=planned;built
	State string `json:"state,omitempty"`
	// +kubebuilder:validation:Optional
	BluePrint string `json:"blueprint,omitempty"`
	// +kubebuilder:validation:Optional
	ConfigValues map[string]ConfigValue `json:"configValues,omitempty"`
	// ConsumerRef can be used to store information about something that is using this structure.
	// +kubebuilder:validation:Optional
	ConsumerRef *corev1.ObjectReference `json:"consumerRef,omitempty"`
	// +kubebuilder:validation:Optional
	Other intstr.IntOrString `json:"other,omitempty"`
	//
	// utility job name, can only be shen when state == built and not job
}

// StructureStatus defines the observed state of the Structure
type StructureStatus struct {
	State               string                 `json:"state,omitempty"`
	BluePrint           string                 `json:"blueprint,omitempty"`
	ConfigValues        map[string]ConfigValue `json:"configValues,omitempty"`
	Job                 *JobStatus             `json:"job,omitempty"`
	Hostname            string                 `json:"hostname,omitempty"`
	Foundation          string                 `json:"foundation,omitempty"`
	FoundationBluePrint string                 `json:"foundationBluePrint,omitempty"`
	// utility job name, utility job result, clear name and result when utility job name is blanked in the spec, the status will be in job Status - will auto clear when the job is complete, also emit events when job is set, started, finishes, etc
}

// JobStatus defines the observed state of the Job
type JobStatus struct {
	State            string `json:"state,omitempty"`
	Script           string `json:"script,omitempty"`
	Message          string `json:"message,omitempty"`
	CanStart         string `json:"canstart,omitempty"`
	Created          string `json:"created,omitempty"`
	LastUpdated      string `json:"lastupdated,omitempty"` // updated seems to be tweeked with some where, so this is named LastUpdated
	Progress         string `json:"progress,omitempty"`
	MaxTimeRemaining string `json:"maxTimeRemaining,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:JSONPath=`.spec.id`,name="Structure",type=integer
// +kubebuilder:printcolumn:JSONPath=`.spec.state`,name="Target State",type=string
// +kubebuilder:printcolumn:JSONPath=`.status.hostname`,name="Hostname",type=string
// +kubebuilder:printcolumn:JSONPath=`.status.foundation`,name="Foundation",type=string
// +kubebuilder:printcolumn:JSONPath=`.status.state`,name="Current State",type=string
// +kubebuilder:printcolumn:JSONPath=`.status.job.state`,name="Job State",type=string

// Structure is the Schema for the structures API
type Structure struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StructureSpec   `json:"spec,omitempty"`
	Status StructureStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// StructureList contains a list of Structure
type StructureList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Structure `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Structure{}, &StructureList{})
}
