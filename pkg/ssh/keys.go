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

package ssh

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"runtime"

	gossh "golang.org/x/crypto/ssh"
)

// NewKeyPair generates a new SSH keypair. This will return a private & public key encoded as DER.
func NewKeyPair() (keyPair *KeyPair, err error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, ErrKeyGeneration
	}

	if err := priv.Validate(); err != nil {
		return nil, ErrValidation
	}

	privDer := x509.MarshalPKCS1PrivateKey(priv)
	privateKey := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Headers: nil, Bytes: privDer})
	pubSSH, err := gossh.NewPublicKey(&priv.PublicKey)
	if err != nil {
		return nil, ErrPublicKey
	}

	return &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  gossh.MarshalAuthorizedKey(pubSSH),
	}, nil
}

// KeyPair represents a Public and Private keypair.
type KeyPair struct {
	PrivateKey []byte
	PublicKey  []byte
}

// ReadFromFile reads a keypair from files.
func (kp *KeyPair) ReadFromFile(privateKeyPath string, publicKeyPath string) error {
	b, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return err
	}
	kp.PrivateKey = b

	b, err = os.ReadFile(publicKeyPath)
	if err != nil {
		return err
	}
	kp.PublicKey = b

	return nil
}

// WriteToFile writes a keypair to files
func (kp *KeyPair) WriteToFile(privateKeyPath string, publicKeyPath string) error {
	files := []struct {
		File  string
		Type  string
		Value []byte
	}{
		{
			File:  privateKeyPath,
			Value: kp.PrivateKey,
		},
		{
			File:  publicKeyPath,
			Value: kp.PublicKey,
		},
	}

	for _, v := range files {
		f, err := os.Create(v.File)
		if err != nil {
			return ErrUnableToWriteFile
		}

		if _, err := f.Write(v.Value); err != nil {
			return ErrUnableToWriteFile
		}

		// windows does not support chmod
		switch runtime.GOOS {
		case "darwin", "linux":
			if err := f.Chmod(0600); err != nil {
				return err
			}
		}
	}

	return nil
}

// Fingerprint calculates the fingerprint of the public key
func (kp *KeyPair) Fingerprint() (string, error) {
	b, _ := base64.StdEncoding.DecodeString(string(kp.PublicKey))
	h := md5.New()

	_, err := io.WriteString(h, string(b))

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
