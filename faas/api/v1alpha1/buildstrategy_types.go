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

const (
	Domain    = "faas.czlingo.io"
	LabelName = Domain + "/name"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BuildStrategySpec defines the desired state of BuildStrategy
type BuildStrategySpec struct {
	Steps     []Step           `json:"steps,omitempty"`
	Parameter []Parameter      `json:"params,omitempty"`
	Volumes   []StrategyVolume `json:"volumes,omitempty"`
}

type Step struct {
	// +required
	Name string `json:"name,omitempty"`

	// +required
	Image string `json:"image,omitempty"`

	// +optional
	Script string `json:"script,omitempty"`

	// +required
	// +patchMergeKey=name
	// +patchStrategy=merge
	Env []corev1.EnvVar `json:"env,omitempty" patchStrategy:"merge" patchMergeKey:"name"`

	// +optional
	// +patchMergeKey=mountPath
	// +patchStrategy=merge
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty" patchStrategy:"merge" patchMergeKey:"mountPath"`

	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
}

type ParameterType string

const (
	ParameterTypeString ParameterType = "string"
	ParameterTypeArray  ParameterType = "array"
)

type Parameter struct {
	// +required
	Name string `json:"name,omitempty"`

	// +optional
	Description string `json:"description,omitempty"`

	// +required
	Type ParameterType `json:"type,omitempty"`

	// +optional
	Default *string `json:"default,omitempty"`

	// +optional
	Defaults *[]string `json:"defaults,omitempty"`
}

type StrategyVolume struct {
	// +required
	Name string `json:"name,omitempty"`

	// +optional
	Description string `json:"description,omitempty"`

	// required
	corev1.VolumeSource `json:",inline"`
}

// BuildStrategyStatus defines the observed state of BuildStrategy
type BuildStrategyStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// BuildStrategy is the Schema for the buildstrategies API
type BuildStrategy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BuildStrategySpec   `json:"spec,omitempty"`
	Status BuildStrategyStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BuildStrategyList contains a list of BuildStrategy
type BuildStrategyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BuildStrategy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BuildStrategy{}, &BuildStrategyList{})
}
