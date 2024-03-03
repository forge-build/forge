// Copyright 2015 Apcera Inc. All rights reserved.

package ssh

import (
	"testing"

	cssh "golang.org/x/crypto/ssh"
)

const password = "password123"

func requireMockedClient() SSHClient {
	c := SSHClient{}
	c.Creds = &Credentials{}
	dial = func(p string, a string, c *cssh.ClientConfig) (*cssh.Client, error) {
		return nil, nil
	}
	readPrivateKey = func(path string) (cssh.AuthMethod, error) {
		return nil, nil
	}
	return c
}

// TestConnectNoUsername tests that an error is returned if no username is provided.
func TestConnectNoUsername(t *testing.T) {
	c := requireMockedClient()
	err := c.Connect()
	if err != ErrInvalidUsername {
		t.Logf("Invalid error type returned %s", err)
		t.Fail()
	}
}

// TestConnectNoPassword tests that an error is returned if no password or key is provided.
func TestConnectNoPassword(t *testing.T) {
	c := requireMockedClient()
	c.Creds.SSHUser = "foo"
	err := c.Connect()
	if err != ErrInvalidAuth {
		t.Logf("Invalid error type returned %s", err)
		t.Fail()
	}
}

// TestConnectAuthPrecedence tests that key based auth takes precedence over password based auth
func TestConnectAuthPrecedence(t *testing.T) {
	c := requireMockedClient()
	count := 0

	c.Creds = &Credentials{
		SSHUser:       "test",
		SSHPassword:   "test",
		SSHPrivateKey: "/foo",
	}

	readPrivateKey = func(path string) (cssh.AuthMethod, error) {
		count++
		return nil, nil
	}
	err := c.Connect()
	if err != nil {
		t.Logf("Expected nil error, got %s", err)
		t.Fail()
	}
	if count != 1 {
		t.Logf("Should have called the password key method 1 time, called %d times", count)
		t.Fail()
	}
}

// TestSetSSHPrivateKey tests the SetSSHPrivateKey method of SSHClient.
func TestSetSSHPrivateKey(t *testing.T) {
	c := requireMockedClient()
	privateKey := "/path/to/private/key"
	c.SetSSHPrivateKey(privateKey)

	if c.Creds.SSHPrivateKey != privateKey {
		t.Errorf("SetSSHPrivateKey failed: expected %s, got %s", privateKey, c.Creds.SSHPrivateKey)
	}
} // ...

// TestGetSSHPrivateKey tests the GetSSHPrivateKey method of SSHClient.
func TestGetSSHPrivateKey(t *testing.T) {
	c := requireMockedClient()
	privateKey := "/path/to/private/key"
	c.Creds.SSHPrivateKey = privateKey

	result := c.GetSSHPrivateKey()

	if result != privateKey {
		t.Errorf("GetSSHPrivateKey failed: expected %s, got %s", privateKey, result)
	}
} // TestSetSSHPassword tests the SetSSHPassword method of SSHClient.
func TestSetSSHPassword(t *testing.T) {
	c := requireMockedClient()
	c.SetSSHPassword(password)

	if c.Creds.SSHPassword != password {
		t.Errorf("SetSSHPassword failed: expected %s, got %s", password, c.Creds.SSHPassword)
	}
} // ...

// TestGetSSHPassword tests the GetSSHPassword method of SSHClient.
func TestGetSSHPassword(t *testing.T) {
	c := requireMockedClient()
	c.Creds.SSHPassword = password

	result := c.GetSSHPassword()

	if result != password {
		t.Errorf("GetSSHPassword failed: expected %s, got %s", password, result)
	}
}

// ...

// TestValidate tests the Validate method of SSHClient.
func TestValidate(t *testing.T) {
	c := requireMockedClient()

	// Test case 1: Empty SSHUser
	c.Creds.SSHUser = ""
	err := c.Validate()
	if err != ErrInvalidUsername {
		t.Errorf("Invalid error type returned %s", err)
	}

	// Test case 2: Empty SSHPassword and SSHPrivateKey
	c.Creds.SSHUser = "test"
	c.Creds.SSHPassword = ""
	c.Creds.SSHPrivateKey = ""
	err = c.Validate()
	if err != ErrInvalidAuth {
		t.Errorf("Invalid error type returned %s", err)
	}

	// Test case 3: Valid credentials
	c.Creds.SSHPassword = "password123"
	err = c.Validate()
	if err != nil {
		t.Errorf("Expected nil error, got %s", err)
	}
}
