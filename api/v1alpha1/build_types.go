/*
Copyright 2024 Forge.

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
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	builderror "github.com/forge-build/forge/pkg/errors"
)

const (
	// BuildFinalizer is the finalizer used by the Build controller to
	// cleanup the build resources when a Build is being deleted.
	BuildFinalizer = "build.forge.build"
)

// BuildSpec defines the desired state of Build
type BuildSpec struct {
	// Paused can be used to prevent controllers from processing the Cluster and all its associated objects.
	// +optional
	Paused bool `json:"paused,omitempty"`

	// Connector is the connector to the infrastructure machine
	// e.g., connector: {type: "ssh", credentials: {name: "aws-credentials", namespace: "default"}}
	// +kubebuilder:validation:Required
	Connector ConnectorSpec `json:"connector"`

	// InfrastructureRef is a reference to the infrastructure object which contains the types of machines to build.
	// e.g. infrastructureRef: {kind: "AWSBuild", name: "ubuntu-2204"}
	// +kubebuilder:validation:Required
	InfrastructureRef *corev1.ObjectReference `json:"infrastructureRef"`

	// Provisioners is a list of provisioners to run on the infrastructure machine
	// +optional
	Provisioners []ProvisionerSpec `json:"provisioners,omitempty"`

	// DeleteCascade is a flag to specify whether the built image(s)
	// going to be cleaned up when the build is deleted.
	// +optional
	DeleteCascade bool `json:"deleteCascade,omitempty"`
}

// ConnectorSpec defines the connector to the infrastructure machine
type ConnectorSpec struct {
	// Type is the type of connector to the infrastructure machine.
	// e.g., type: "ssh"
	Type string `json:"type"`

	// Credentials is a reference to the secret containing the credentials to connect to the infrastructure machine
	// The secret should contain the following
	// - username
	// - password and/or privateKey
	// - host
	Credentials *corev1.SecretReference `json:"credentials,omitempty"`
}

// ProvisionerSpec defines the provisioner to run on the infrastructure machine
type ProvisionerSpec struct {
	// Type is the type of provisioner to run on the infrastructure machine
	// e.g., type: "builtin" or type: "external"
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=built-in/shell;external
	Type ProvisionerType `json:"type"`

	// AllowFail is a flag to allow the provisioner to fail
	// +optional
	AllowFail bool `json:"allowFail,omitempty"`

	// Run is the command to run on the infrastructure machine
	// +optional
	Run *string `json:"run,omitempty"`

	// Ref is a reference to the provisioner object which contains the types of provisioners to run.
	Ref *corev1.ObjectReference `json:"ref,omitempty"`
}

type ProvisionerType string

const (
	ProvisionerTypeShell    ProvisionerType = "built-in/shell"
	ProvisionerTypeExternal ProvisionerType = "external"
)

// BuildPhase BuildStatus defines the observed state of Build
type BuildPhase string

const (
	BuildPhasePending     BuildPhase = "Pending"
	BuildPhaseBuilding    BuildPhase = "Building"
	BuildPhaseTerminating BuildPhase = "Terminating"
	BuildPhaseCompleted   BuildPhase = "Completed"
	BuildPhaseFailed      BuildPhase = "Failed"
	BuildPhaseUnknown     BuildPhase = "Unknown"
)

type BuildStatus struct {
	// FailureDomains is a slice of failure domain objects synced from the infrastructure provider.
	// +optional
	FailureDomains FailureDomains `json:"failureDomains,omitempty"`

	// FailureReason indicates that there is a fatal problem reconciling the
	// state, and will be set to a token value suitable for
	// programmatic interpretation.
	// +optional
	FailureReason *builderror.BuildStatusError `json:"failureReason,omitempty"`

	// FailureMessage indicates that there is a fatal problem reconciling the
	// state, and will be set to a descriptive error message.
	// +optional
	FailureMessage *string `json:"failureMessage,omitempty"`

	// Conditions define the current service state of the cluster.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`

	// InfrastructureReady is the state of the machine, which will be seted to true after it successfully in running state
	//+optional
	InfrastructureReady bool `json:"infrastructureReady,omitempty"`

	// Connected describes if the connection to the underlying infrastructure machine has been established
	//+optional
	Connected bool `json:"connected,omitempty"`

	// ProvisionersReady describes the state of provisioners for the Build
	// once all provisioners have finished successfully, this will be true
	//+optional
	ProvisionersReady bool `json:"provisionersReady,omitempty"`

	// Build Phase which is used to track the state of the build process
	// E.g. Pending, Building, Terminating, Failed etc.
	//+optional
	Phase string `json:"phase,omitempty"`

	// Ready is the state of the build process, true if machine image is ready, false if not
	//+optional
	Ready bool `json:"ready,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=builds,scope=Namespaced,categories=forge,singular=build
//+kubebuilder:printcolumn:name="Build",type="string",JSONPath=".metadata.labels['forge\\.build\\.io/build-name']",description="Build"
//+kubebuilder:printcolumn:name="Machine Ready",type="string",JSONPath=".status.machineReady",description="Machine Ready"
//+kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.ready",description="Build is ready"

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

// GetConditions returns the set of conditions for this object.
func (c *Build) GetConditions() clusterv1.Conditions {
	return c.Status.Conditions
}

// SetConditions sets the conditions on this object.
func (c *Build) SetConditions(conditions clusterv1.Conditions) {
	c.Status.Conditions = conditions
}

func init() {
	objectTypes = append(objectTypes, &Build{}, &BuildList{})
}

// FailureDomains is a slice of FailureDomains.
type FailureDomains map[string]FailureDomainSpec

// FilterControlPlane returns a FailureDomain slice containing only the domains suitable to be used
// for control plane nodes.
func (in FailureDomains) FilterControlPlane() FailureDomains {
	res := make(FailureDomains)
	for id, spec := range in {
		if spec.Infrastructure {
			res[id] = spec
		}
	}
	return res
}

// GetIDs returns a slice containing the ids for failure domains.
func (in FailureDomains) GetIDs() []*string {
	ids := make([]*string, 0, len(in))
	for id := range in {
		ids = append(ids, ptr.To(id))
	}
	return ids
}

// FailureDomainSpec is the Schema for Forge API failure domains.
// It allows controllers to understand how many failure domains a build can optionally span across.
type FailureDomainSpec struct {
	// Infrastructure determines if this failure domain is suitable for use by infrastructure machines.
	// +optional
	Infrastructure bool `json:"controlPlane,omitempty"`

	// Attributes is a free form map of attributes an infrastructure provider might use or require.
	// +optional
	Attributes map[string]string `json:"attributes,omitempty"`
}

// ANCHOR_END: ClusterStatus

// SetTypedPhase sets the Phase field to the string representation of ClusterPhase.
func (c *BuildStatus) SetTypedPhase(p BuildPhase) {
	c.Phase = string(p)
}

// GetTypedPhase attempts to parse the Phase field and return
// the typed ClusterPhase representation as described in `machine_phase_types.go`.
func (c *BuildStatus) GetTypedPhase() BuildPhase {
	switch phase := BuildPhase(c.Phase); phase {
	case
		BuildPhasePending,
		BuildPhaseBuilding,
		BuildPhaseTerminating,
		BuildPhaseCompleted,
		BuildPhaseFailed:
		return phase
	default:
		return BuildPhaseUnknown
	}
}
