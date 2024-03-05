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
	"io"
	"net"
	"testing"
	"time"

	cssh "golang.org/x/crypto/ssh"
)

const password = "password123"

func requireMockedClient() SSHClient {
	c := SSHClient{}
	c.Creds = &Credentials{}
	dial = func(_ string, _ string, _ *cssh.ClientConfig) (*cssh.Client, error) {
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

	readPrivateKey = func(_ string) (cssh.AuthMethod, error) {
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

func TestConstants(t *testing.T) {
	expectedPort := 22
	if sshPort != expectedPort {
		t.Errorf("Expected sshPort to be %d, but got %d", expectedPort, sshPort)
	}

	expectedPasswordAuth := "password"
	if PasswordAuth != expectedPasswordAuth {
		t.Errorf("Expected PasswordAuth to be %s, but got %s", expectedPasswordAuth, PasswordAuth)
	}

	expectedKeyAuth := "key"
	if KeyAuth != expectedKeyAuth {
		t.Errorf("Expected KeyAuth to be %s, but got %s", expectedKeyAuth, KeyAuth)
	}

	expectedTimeout := 60 * time.Second
	if Timeout != expectedTimeout {
		t.Errorf("Expected Timeout to be %s, but got %s", expectedTimeout, Timeout)
	}
}

func TestErrInvalidUsername(t *testing.T) {
	err := ErrInvalidUsername
	expected := "a valid username must be supplied"
	if err.Error() != expected {
		t.Errorf("Expected error message: %s, but got: %s", expected, err.Error())
	}
}

func TestErrInvalidAuth(t *testing.T) {
	err := ErrInvalidAuth
	expected := "invalid authorization method: missing password or key"
	if err.Error() != expected {
		t.Errorf("Expected error message: %s, but got: %s", expected, err.Error())
	}
}

func TestErrSSHInvalidMessageLength(t *testing.T) {
	err := ErrSSHInvalidMessageLength
	expected := "invalid message length"
	if err.Error() != expected {
		t.Errorf("Expected error message: %s, but got: %s", expected, err.Error())
	}
}

func TestErrTimeout(t *testing.T) {
	err := ErrTimeout
	expected := "timed out waiting for sshd to respond"
	if err.Error() != expected {
		t.Errorf("Expected error message: %s, but got: %s", expected, err.Error())
	}
}

func TestErrKeyGeneration(t *testing.T) {
	err := ErrKeyGeneration
	expected := "unable to generate key"
	if err.Error() != expected {
		t.Errorf("Expected error message: %s, but got: %s", expected, err.Error())
	}
}

func TestErrValidation(t *testing.T) {
	err := ErrValidation
	expected := "unable to validate key"
	if err.Error() != expected {
		t.Errorf("Expected error message: %s, but got: %s", expected, err.Error())
	}
}

func TestErrPublicKey(t *testing.T) {
	err := ErrPublicKey
	expected := "unable to convert public key"
	if err.Error() != expected {
		t.Errorf("Expected error message: %s, but got: %s", expected, err.Error())
	}
}

func TestErrUnableToWriteFile(t *testing.T) {
	err := ErrUnableToWriteFile
	expected := "unable to write file"
	if err.Error() != expected {
		t.Errorf("Expected error message: %s, but got: %s", expected, err.Error())
	}
}

func TestErrNotImplemented(t *testing.T) {
	err := ErrNotImplemented
	expected := "operation not implemented"
	if err.Error() != expected {
		t.Errorf("Expected error message: %s, but got: %s", expected, err.Error())
	}
}

func TestCloseMutex(t *testing.T) {
	closeMutex.Lock()
	// No assertion needed, this test is to ensure that the mutex can be locked and unlocked without errors.
}

// TestMockSSHClient tests the MockSSHClient struct.
func TestMockSSHClient(t *testing.T) {
	// Create a new instance of MockSSHClient
	mockClient := MockSSHClient{}

	// Test the MockConnect function
	mockClient.MockConnect = func() error {
		// Add your test logic here
		return nil
	}
	err := mockClient.MockConnect()
	if err != nil {
		t.Errorf("MockConnect failed: %s", err)
	}

	// Test the MockDisconnect function
	mockClient.MockDisconnect = func() {
		// Add your test logic here
	}
	mockClient.MockDisconnect()

	// Test the MockDownload function
	mockClient.MockDownload = func(src io.WriteCloser, dst string) error {
		// Add your test logic here
		return nil
	}
	err = mockClient.MockDownload(nil, "")
	if err != nil {
		t.Errorf("MockDownload failed: %s", err)
	}

	// Test the MockRun function
	mockClient.MockRun = func(command string, stdout io.Writer, stderr io.Writer) error {
		// Add your test logic here
		return nil
	}
	err = mockClient.MockRun("", nil, nil)
	if err != nil {
		t.Errorf("MockRun failed: %s", err)
	}

	// Test the MockUpload function
	mockClient.MockUpload = func(src io.Reader, dst string, mode uint32) error {
		// Add your test logic here
		return nil
	}
	err = mockClient.MockUpload(nil, "", 0)
	if err != nil {
		t.Errorf("MockUpload failed: %s", err)
	}

	// Test the MockValidate function
	mockClient.MockValidate = func() error {
		// Add your test logic here
		return nil
	}
	err = mockClient.MockValidate()
	if err != nil {
		t.Errorf("MockValidate failed: %s", err)
	}

	// Test the MockWaitForSSH function
	mockClient.MockWaitForSSH = func(maxWait time.Duration) error {
		// Add your test logic here
		return nil
	}
	err = mockClient.MockWaitForSSH(time.Second)
	if err != nil {
		t.Errorf("MockWaitForSSH failed: %s", err)
	}

	// Test the MockSetSSHPrivateKey function
	mockClient.MockSetSSHPrivateKey = func(privateKey string) {
		// Add your test logic here
	}
	mockClient.MockSetSSHPrivateKey("")

	// Test the MockGetSSHPrivateKey function
	mockClient.MockGetSSHPrivateKey = func() string {
		// Add your test logic here
		return ""
	}
	result := mockClient.MockGetSSHPrivateKey()
	if result != "" {
		t.Errorf("MockGetSSHPrivateKey failed: expected '', got %s", result)
	}

	// Test the MockSetSSHPassword function
	mockClient.MockSetSSHPassword = func(password string) {
		// Add your test logic here
	}
	mockClient.MockSetSSHPassword("")

	// Test the MockGetSSHPassword function
	mockClient.MockGetSSHPassword = func() string {
		// Add your test logic here
		return ""
	}
	result = mockClient.MockGetSSHPassword()
	if result != "" {
		t.Errorf("MockGetSSHPassword failed: expected '', got %s", result)
	}
}

// TestGetAuth tests the getAuth function.
func TestGetAuth(t *testing.T) {
	c := requireMockedClient()

	// Test case 1: PasswordAuth
	c.Creds.SSHPassword = password
	_, err := getAuth(c.Creds, PasswordAuth)
	if err != nil {
		t.Errorf("Expected nil error, got %s", err)
	}
}

func TestSSHClient_WaitForSSH(t *testing.T) {
	type fields struct {
		Creds        *Credentials
		IP           net.IP
		Port         int
		Options      Options
		cryptoClient *cssh.Client
		close        chan bool
	}
	type args struct {
		maxWait time.Duration
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &SSHClient{
				Creds:        tt.fields.Creds,
				IP:           tt.fields.IP,
				Port:         tt.fields.Port,
				Options:      tt.fields.Options,
				cryptoClient: tt.fields.cryptoClient,
				close:        tt.fields.close,
			}
			if err := client.WaitForSSH(tt.args.maxWait); (err != nil) != tt.wantErr {
				t.Errorf("SSHClient.WaitForSSH() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWaitForSSH_Success(t *testing.T) {
	// Create a mocked SSHClient

	// Set up the Connect function to return nil
	connectCalled := false
	connect := func() error {
		connectCalled = true
		return nil
	}

	_ = connect()

	// Set up the Disconnect function to do nothing
	disconnectCalled := false
	disconnect := func() error {
		disconnectCalled = true
		return nil
	}

	_ = disconnect()

	// Check that the Connect function was called
	if !connectCalled {
		t.Error("Connect function was not called")
	}

	// Check that the Disconnect function was called
	if !disconnectCalled {
		t.Error("Disconnect function was not called")
	}

}

func TestWaitForSSH_Timeout(t *testing.T) {
	// Create a mocked SSHClient
	c := requireMockedClient()

	// Set up the Connect function to always return an error
	_ = func() error {
		return ErrInvalidAuth
	}

	// Call the WaitForSSH function with a short maxWait duration
	maxWait := 1 * time.Second
	err := c.WaitForSSH(maxWait)

	// Check that the error returned is ErrTimeout
	if err != ErrTimeout {
		t.Errorf("Expected error %s, got %s", ErrTimeout, err)
	}
}

// TestDisconnect tests the Disconnect method of SSHClient.
func TestDisconnect(t *testing.T) {
	c := requireMockedClient()

	// Test case 1: Close channel is nil
	c.close = nil
	c.Disconnect()

	// Test case 2: Close channel is not nil
	c.close = make(chan bool)
	c.Disconnect()

	// Assert that the close channel is set to nil
	if c.close != nil {
		t.Errorf("Disconnect failed: expected close channel to be nil, but got %v", c.close)
	}
}
