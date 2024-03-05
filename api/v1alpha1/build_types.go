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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	builderror "github.com/forge-build/forge/pkg/errors"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BuildSpec defines the desired state of Build
type BuildSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// InfrastructureRef is a reference to the infrastructure object which contains the types of machines to build.
	// for e.g infrastructureRef: {kind: "AWSBuild", name: "ubuntu-2204"}
	InfrastructureRef *corev1.ObjectReference `json:"infrastructureRef"`
}

// BuildStatus defines the observed state of Build
type BuildPhase string

const (
	PhasePending     BuildPhase = "Pending"
	PhaseBuilding    BuildPhase = "Building"
	PhaseTerminating BuildPhase = "Terminating"
	PhaseCompleted   BuildPhase = "Completed"
	PhaseFailed      BuildPhase = "Failed"
)

type BuildStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// FailureReason indicates that there is a fatal problem reconciling the
	// state, and will be set to a token value suitable for
	// programmatic interpretation.
	// +optional
	FailureReason *builderror.BuildStatusError `json:"failureReason,omitempty"`

	// FailureMessage indicates that there is a fatal problem reconciling the
	// state, and will be set to a descriptive error message.
	// +optional
	FailureMessage *string `json:"failureMessage,omitempty"`

	// MachineReady is the state of the machine, which will be seted to true after it successfully in running state
	//+optional
	InfrastructureReady *bool `json:"infrastructureReady,omitempty"`

	// Connected describes if the connection to the underlying infrastructure machine has been established
	//+optional
	Connected *bool `json:"connected,omitempty"`

	// ProvisionersReady describes the state of provisioners for the Build
	// once all provisioners has finished successfully this will be true
	//+optional
	ProvisionersReady *bool `json:"provisionersReady,omitempty"`

	// Build Phase which is used to track the state of the build process
	Phase BuildPhase `json:"phase,omitempty"`

	// Ready is the state of the build process, true if machine image is ready, false if not
	//+optional
	Ready *bool `json:"ready,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Build is the Schema for the builds API
type Build struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BuildSpec   `json:"spec,omitempty"`
	Status BuildStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BuildList contains a list of Build
type BuildList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Build `json:"items"`
}

func init() {
	objectTypes = append(objectTypes, &Build{}, &BuildList{})
}
