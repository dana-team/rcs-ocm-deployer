/*
Copyright 2022.

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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RCSConfigSpec defines the desired state of RCSConfig
type RCSConfigSpec struct {
	// PlacementsNamespace defines the namespace where the Placement CRs exist
	// +kubebuilder:default:="placements"
	PlacementsNamespace string `json:"placementsNamespace"`

	// Placements is an array of Placement names that the operator should use
	Placements []string `json:"placements"`

	// DefaultResources is the default resources to be assigned to Capp.
	// If other resources are specified then they override the default values.
	DefaultResources corev1.ResourceRequirements `json:"defaultResources"`

	// InvalidHostnamePatterns is an optional slice of regex patterns to be used to validate the hostname of the Capp.
	// If the Capp hostname matches a pattern, it is blocked from being created.
	// +kubebuilder:default:={}
	InvalidHostnamePatterns []string `json:"invalidHostnamePatterns"`
}

// RCSConfigStatus defines the observed state of RCSConfig
type RCSConfigStatus struct{}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// RCSConfig is the Schema for the rcsconfigs API
type RCSConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RCSConfigSpec   `json:"spec,omitempty"`
	Status RCSConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RCSConfigList contains a list of RCSConfig
type RCSConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RCSConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RCSConfig{}, &RCSConfigList{})
}
