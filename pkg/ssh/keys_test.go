package ssh

import (
	"bytes"
	"crypto/md5"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"runtime"
	"testing"
)

func TestKeyPairFingerprint(t *testing.T) {
	// Create a KeyPair instance with a sample public key
	publicKey := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDZz6qz5J1z3z7XQz8R..."
	keyPair := KeyPair{
		PublicKey: []byte(publicKey),
	}

	// Calculate the expected fingerprint
	b, _ := base64.StdEncoding.DecodeString(publicKey)
	h := md5.New()
	_, err := io.WriteString(h, string(b))
	if err != nil {
		t.Errorf("Error writing to hash: %s", err)
	}
	expectedFingerprint := fmt.Sprintf("%x", h.Sum(nil))

	// Call the Fingerprint method
	fingerprint, err := keyPair.Fingerprint()
	if err != nil {
		t.Errorf("Error calculating fingerprint: %s", err)
	}

	// Compare the actual fingerprint with the expected fingerprint
	if fingerprint != expectedFingerprint {
		t.Errorf("Fingerprint mismatch. Expected: %s, Got: %s", expectedFingerprint, fingerprint)
	}
}

func TestKeyPairWriteToFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a KeyPair instance with sample private and public keys
	privateKey := []byte("sample private key")
	publicKey := []byte("sample public key")
	keyPair := KeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}

	// Define the expected file paths
	privateKeyPath := tempDir + "/private_key"
	publicKeyPath := tempDir + "/public_key"

	// Call the WriteToFile method
	err := keyPair.WriteToFile(privateKeyPath, publicKeyPath)
	if err != nil {
		t.Errorf("Error writing key pair to file: %s", err)
	}

	// Check if the private key file exists
	_, err = os.Stat(privateKeyPath)
	if err != nil {
		t.Errorf("Private key file does not exist: %s", err)
	}

	// Check if the public key file exists
	_, err = os.Stat(publicKeyPath)
	if err != nil {
		t.Errorf("Public key file does not exist: %s", err)
	}

	// Check file permissions on Unix-like systems
	switch runtime.GOOS {
	case "darwin", "linux":
		// Check private key file permissions
		privateKeyInfo, err := os.Stat(privateKeyPath)
		if err != nil {
			t.Errorf("Error getting private key file info: %s", err)
		}
		privateKeyPerm := privateKeyInfo.Mode().Perm()
		if privateKeyPerm != 0600 {
			t.Errorf("Private key file has incorrect permissions. Expected: 0600, Got: %o", privateKeyPerm)
		}

		// Check public key file permissions
		publicKeyInfo, err := os.Stat(publicKeyPath)
		if err != nil {
			t.Errorf("Error getting public key file info: %s", err)
		}
		publicKeyPerm := publicKeyInfo.Mode().Perm()
		if publicKeyPerm != 0600 {
			t.Errorf("Public key file has incorrect permissions. Expected: 0600, Got: %o", publicKeyPerm)
		}
	}
}
func TestKeyPairReadFromFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create temporary private and public key files
	privateKeyPath := tempDir + "/private_key"
	publicKeyPath := tempDir + "/public_key"
	privateKeyContent := []byte("sample private key")
	publicKeyContent := []byte("sample public key")
	err := os.WriteFile(privateKeyPath, privateKeyContent, 0600)
	if err != nil {
		t.Fatalf("Failed to create temporary private key file: %s", err)
	}
	err = os.WriteFile(publicKeyPath, publicKeyContent, 0600)
	if err != nil {
		t.Fatalf("Failed to create temporary public key file: %s", err)
	}

	// Create a KeyPair instance
	keyPair := KeyPair{}

	// Call the ReadFromFile method
	err = keyPair.ReadFromFile(privateKeyPath, publicKeyPath)
	if err != nil {
		t.Errorf("Error reading key pair from file: %s", err)
	}

	// Compare the private key content
	if !bytes.Equal(keyPair.PrivateKey, privateKeyContent) {
		t.Errorf("Private key content mismatch. Expected: %s, Got: %s", privateKeyContent, keyPair.PrivateKey)
	}

	// Compare the public key content
	if !bytes.Equal(keyPair.PublicKey, publicKeyContent) {
		t.Errorf("Public key content mismatch. Expected: %s, Got: %s", publicKeyContent, keyPair.PublicKey)
	}
}
func TestNewKeyPair(t *testing.T) {
	keyPair, err := NewKeyPair()
	if err != nil {
		t.Errorf("Error generating key pair: %s", err)
	}

	// Validate the private key
	block, _ := pem.Decode(keyPair.PrivateKey)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		t.Errorf("Invalid private key format")
	}
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		t.Errorf("Error parsing private key: %s", err)
	}
	if err := priv.Validate(); err != nil {
		t.Errorf("Private key validation failed: %s", err)
	}
}
