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
	"crypto/x509"
	"encoding/pem"

	corev1 "k8s.io/api/core/v1"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
)

// GetCredentialsFromSecret returns the public key from a private key in PEM format.
func GetCredentialsFromSecret(secret *corev1.Secret) (username, password, privateKey, publicKey string) {
	username = string(secret.Data["username"])
	password = string(secret.Data["password"])
	privateKey = string(secret.Data["privateKey"])
	publicKey = string(secret.Data["publicKey"])

	return username, password, privateKey, publicKey
}

func GetPublicKeyFromPrivateKey(privateKeyPem string) (string, error) {
	// Decode the PEM block
	block, _ := pem.Decode([]byte(privateKeyPem))
	if block == nil {
		return "", errors.New("failed to decode PEM block containing private key")
	}

	// Parse the RSA private key
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}

	// Generate the SSH public key
	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", err
	}

	// Convert and return the public key as an authorized keys line
	return string(ssh.MarshalAuthorizedKey(publicKey)), nil
}
