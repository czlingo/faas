/*
Copyright 2023.

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

type Strategy string

const (
	BuildStrategyBuildpacks Strategy = "faas-buildstrategy-buildpacks"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BuildingSpec defines the desired state of Building
type BuildingSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Strategy        Strategy                `json:"strategy,omitempty"`
	Image           string                  `json:"image,omitempty"`
	ImageCredential *corev1.SecretReference `json:"imageCredential,omitempty"`
	SSHCredential   *corev1.SecretReference `json:"sshCredential,omitempty"`
	BuildingImpl    `json:",inline"`
}

type BuildingImpl struct {
	Repo        string                  `json:"repo,omitempty"`
	SubPath     string                  `json:"subpath,omitempty"`
	Revision    *string                 `json:"revision,omitempty"`
	Credentials *corev1.SecretReference `json:"credentials,omitempty"`
}

// BuildingStatus defines the observed state of Building
type BuildingStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	State *State `json:"state,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Building is the Schema for the buildings API
type Building struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BuildingSpec   `json:"spec,omitempty"`
	Status BuildingStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BuildingList contains a list of Building
type BuildingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Building `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Building{}, &BuildingList{})
}
