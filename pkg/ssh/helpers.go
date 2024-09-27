package ssh

import (
	"net"

	corev1 "k8s.io/api/core/v1"
)

func NewSSHClient(secret *corev1.Secret) (*SSHClient, error) {
	creds := &Credentials{
		SSHUser: string(secret.Data["username"]),
	}
	if password, ok := secret.Data["password"]; ok {
		creds.SSHPassword = string(password)
	}
	if privateKey, ok := secret.Data["privateKey"]; ok {
		creds.SSHPrivateKey = string(privateKey)
	}
	ip := net.ParseIP(string(secret.Data["host"]))

	sshClient := &SSHClient{
		Creds: creds,
		IP:    ip,
		Port:  22,
	}

	return sshClient, nil
}
