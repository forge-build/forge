package v1alpha1

import corev1 "k8s.io/api/core/v1"

// ConnectionSpec defines the schema of connector to the infrastructure machine.
type ConnectionSpec struct {
	// Username is the username to connect to the infrastructure machine.
	// +required
	// +kubebuilder:default:="root"
	Username string `json:"username"`

	// CredentialsRef is a reference to the secret which contains the credentials to connect to the infrastructure machine.
	// The secret should contain the following keys:
	// - username: The username to connect to the machine
	// - password: The password for authentication (if applicable)
	// - privateKey: The SSH private key for authentication (if applicable)
	// +optional
	SSHCredentialsRef *corev1.SecretReference `json:"sshCredentialsRef,omitempty"`

	// GenerateSSHKey is a flag to specify whether the controller should generate a new private key for the connection.
	// GenerateSSHKey will take precedence over the privateKey in the secret.
	// +optional
	GenerateSSHKey bool `json:"generateSSHKey,omitempty"`
}
