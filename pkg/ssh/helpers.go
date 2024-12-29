/*
Copyright 2024 The Forge Authors.

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
