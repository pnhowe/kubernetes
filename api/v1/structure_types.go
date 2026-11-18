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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConditionReady is the type of the Ready condition
const ConditionReady = "Ready"

// The reasons for the Ready condition
const (
	// ReasonReconciled is Ready=True, the Structure is in sync
	ReasonReconciled = "Reconciled"
	// ReasonJobRunning is Ready=False, a job is running in contractor
	ReasonJobRunning = "JobRunning"
	// ReasonApplyingChanges is Ready=False, a change was just made, and the operator is waiting to see it take effect
	ReasonApplyingChanges = "ApplyingChanges"
	// ReasonContractorError is Ready=False, contractor could not be reached or returned an error
	ReasonContractorError = "ContractorError"
	// ReasonInvalidSpec is Ready=False, the spec can not be acted on
	ReasonInvalidSpec = "InvalidSpec"
	// ReasonSyncFailed is Ready=False, in follow mode updating the spec to match contractor failed (ie: the webhook rejected it)
	ReasonSyncFailed = "SyncFailed"
)

// The states a Structure can be in
const (
	StatePlanned = "planned"
	StateBuilt   = "built"
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
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	ConfigValues ConfigValues `json:"configValues,omitempty"`
	// ConsumerRef can be used to store information about something that is using this structure.
	// +kubebuilder:validation:Optional
	ConsumerRef *corev1.ObjectReference `json:"consumerRef,omitempty"`
}

// StructureStatus defines the observed state of the Structure
type StructureStatus struct {
	State     string `json:"state,omitempty"`
	BluePrint string `json:"blueprint,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	ConfigValues        ConfigValues `json:"configValues,omitempty"`
	Job                 *JobStatus   `json:"job,omitempty"`
	Hostname            string       `json:"hostname,omitempty"`
	Foundation          string       `json:"foundation,omitempty"`
	FoundationBluePrint string       `json:"foundationBluePrint,omitempty"`
	// Conditions has "Ready", which is True when the Structure matches the spec (or contractor, in follow mode)
	// and nothing is running, see the Reason for what it is waiting on or what is wrong.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
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
// +kubebuilder:printcolumn:JSONPath=`.status.conditions[?(@.type=="Ready")].status`,name="Ready",type=string
// +kubebuilder:printcolumn:JSONPath=`.status.conditions[?(@.type=="Ready")].reason`,name="Reason",type=string,priority=1

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
