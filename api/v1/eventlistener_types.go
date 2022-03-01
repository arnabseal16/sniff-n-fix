/*
Copyright 2021.

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
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/clock"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// EventListenerSpec defines the desired state of EventListener
type EventListenerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Actions []EventListenerAction `json:"actions,omitempty"`
}

// EventListenerStatus defines the observed state of EventListener
type EventListenerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Conditions []StatusCondition `json:"conditions,omitempty"`
}

var Clock clock.Clock = clock.RealClock{}

func (els *EventListenerStatus) SetCondition(conditionType ConditionType, status ConditionStatus, reason, message string) {
	newCondition := StatusCondition{
		Type:    conditionType,
		Status:  status,
		Reason:  reason,
		Message: message,
	}

	nowTime := metav1.NewTime(Clock.Now())
	newCondition.LastTransitionTime = &nowTime

	// Search through existing conditions
	for idx, cond := range els.Conditions {
		// Skip unrelated conditions
		if cond.Type != conditionType {
			continue
		}

		// If this update doesn't contain a state transition, we don't update
		// the conditions LastTransitionTime to Now()
		if cond.Status == status {
			newCondition.LastTransitionTime = cond.LastTransitionTime
		}

		// Overwrite the existing condition
		els.Conditions[idx] = newCondition
		return
	}

	// If we've not found an existing condition of this type, we simply insert
	// the new condition into the slice.
	els.Conditions = append(els.Conditions, newCondition)
}

type StatusCondition struct {
	// Type of the condition, known values are ('Ready').
	Type ConditionType `json:"type"`

	// Status of the condition, one of ('True', 'False', 'Unknown').
	Status ConditionStatus `json:"status"`

	// LastTransitionTime is the timestamp corresponding to the last status
	// change of this condition.
	// +optional
	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty"`

	// Reason is a brief machine readable explanation for the condition's last
	// transition.
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message is a human readable description of the details of the last
	// transition, complementing reason.
	// +optional
	Message string `json:"message,omitempty"`
}

type EventListenerAction struct {
	ActionType    ActionType   `json:"action_type,omitempty"`
	ResourceType  ResourceType `json:"resource_type,omitempty"`
	Target        string       `json:"target,omitempty"`
	ReceiptHandle *string      `json:receipthandle,omitempty`
}

// ConditionStatus represents a condition's status.
// +kubebuilder:validation:Enum=delete
type ActionType string

// ConditionStatus represents a condition's status.
// +kubebuilder:validation:Enum=pod
type ResourceType string

// ConditionStatus represents a condition's status.
// +kubebuilder:validation:Enum=True;False
type ConditionStatus string

// ConditionStatus represents a condition's status.
// +kubebuilder:validation:Enum=PodDeleted;Unknown
type ConditionType string

// These are valid condition statuses. "ConditionTrue" means a resource is in
// the condition; "ConditionFalse" means a resource is not in the condition;
// "ConditionUnknown" means kubernetes can't decide if a resource is in the
// condition or not. In the future, we could add other intermediate
// conditions, e.g. ConditionDegraded.
const (
	// ActionDelete represents an event action to Delete
	ActionDelete ActionType = "delete"

	// Increase Replica Count
	ActionScaleUp ActionType = "scale-up"

	// Decrease Replica Count
	ActionScaleDown ActionType = "scale-down"

	// ResourcePod represents a Kubernetes Pod type
	ResourcePod ResourceType = "pod"

	// ResourceDep represents a Kubernetes Deployment type
	ResourceDep ResourceType = "deployment"

	// ConditionTrue represents the fact that a given condition is true
	ConditionTrue ConditionStatus = "True"

	// ConditionFalse represents the fact that a given condition is false
	ConditionFalse ConditionStatus = "False"

	ConditionPodDeleted ConditionType = "PodDeleted"

	ConditionUnknown ConditionType = "Unknown"
)

var conditionTypes = [...]ConditionType{
	ConditionPodDeleted,
	ConditionUnknown,
}

func GetConditionType(action EventListenerAction) ConditionType {
	kindTitled := strings.Title(string(action.ResourceType))
	actionTitled := strings.Title(string(action.ActionType))
	for _, t := range conditionTypes {
		conditionStr := string(t)
		if strings.Contains(conditionStr, kindTitled) &&
			strings.Contains(conditionStr, actionTitled) {
			return t
		}
	}
	return ConditionUnknown
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// EventListener is the Schema for the eventlisteners API
type EventListener struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EventListenerSpec   `json:"spec,omitempty"`
	Status EventListenerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EventListenerList contains a list of EventListener
type EventListenerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EventListener `json:"items"`
}

func (el *EventListener) IsBeingDeleted() bool {
	return !el.ObjectMeta.DeletionTimestamp.IsZero()
}

func init() {
	SchemeBuilder.Register(&EventListener{}, &EventListenerList{})
}
