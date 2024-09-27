package ssh

import (
	"crypto/x509"
	"encoding/pem"

	corev1 "k8s.io/api/core/v1"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
)

// GetCredentialsFromSecret returns the public key from a private key in PEM format.
func GetCredentialsFromSecret(secret *corev1.Secret) (username, password, privateKey string) {
	username = string(secret.Data["username"])
	password = string(secret.Data["password"])
	privateKey = string(secret.Data["privateKey"])

	return username, password, privateKey
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
