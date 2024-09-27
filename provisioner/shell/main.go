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

// Package shell package provides a shell command execution interface.
package shell

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/forge-build/forge/pkg/ssh"
)

const (
	CredentialsSecretPath string = "/var/run/secrets/ssh-credentials"

	SSHTimeout = 2 * time.Minute
)

var (
	// Namespace is the namespace where the build is running
	Namespace string
	// ScriptToRun is the script to run
	ScriptToRun string
	// ScriptToRunRef is the name of the configmap containing the script to run
	ScriptToRunRef string
	// SSHCredentialsSecretName is the name of the secret containing the credentials
	SSHCredentialsSecretName string
)

func main() {
	ctrl.SetLogger(klog.Background())
	klog.InitFlags(nil)

	flag.StringVar(&Namespace, "namespace", "forge-core", "The Build namespace")
	flag.StringVar(&ScriptToRun, "run-script", "", "The script to run")
	flag.StringVar(&ScriptToRunRef, "run-script-ref", "", "The name of configmap containing the script to run")
	flag.StringVar(&SSHCredentialsSecretName, "ssh-credentials-secret-name", "", "The name of secret containing the ssh credentials")

	flag.Parse()

	ctrl.SetLogger(klog.NewKlogr())
	logger := ctrl.Log.WithName("shell-provisioner")
	ctx := context.Background()

	k8sClient, err := initClient()
	if err != nil {
		logger.Error(err, "Error creating Kubernetes client")
		klog.Exit(err)
	}

	// Read the secret
	secret := &corev1.Secret{}
	err = k8sClient.Get(ctx, client.ObjectKey{Namespace: Namespace, Name: SSHCredentialsSecretName}, secret)
	if err != nil {
		logger.Error(err, "Error getting secret")
		klog.Exit(err)
	}

	// Read scriptToRunRef
	if ScriptToRunRef != "" {
		cm := &corev1.ConfigMap{}
		if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: Namespace, Name: ScriptToRunRef}, cm); err != nil {
			logger.Error(err, "Error getting configmap")
			klog.Exit(err)
		}
		for _, v := range cm.Data {
			ScriptToRun = v
		}
	}

	err = run(logger, secret)
	if err != nil {
		logger.Error(err, "Error running script")
		klog.Exit(err)
	}

}

func run(logger logr.Logger, secret *corev1.Secret) error {
	sshClient, err := ssh.NewSSHClient(secret)
	if err != nil {
		return errors.Wrap(err, "Error creating SSH client")
	}
	if err := sshClient.WaitForSSH(SSHTimeout); err != nil {
		return errors.Wrap(err, "failed to connect to the machine via ssh")
	}
	defer sshClient.Disconnect()

	script := ScriptToRun
	if script == "" {
		return errors.New("script to run is empty")
	}

	output := &bytes.Buffer{}
	errOutput := &bytes.Buffer{}
	err = sshClient.Run(
		script,
		output,
		errOutput,
	)
	if err != nil {
		logger.Error(err, "Failed to run script", "output", output.String(), "error", errOutput.String())
		return errors.Wrapf(err, "Failed to run script: %s", errOutput.String())
	}

	return nil
}

func initClient() (client.Client, error) {
	// Load the kubeconfig from default location
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	// Create a new scheme
	s := runtime.NewScheme()

	// Add core types to the scheme, you can add more types as needed
	utilruntime.Must(clientgoscheme.AddToScheme(s))
	//buildv1.AddToScheme(scheme))

	// Create a new client with the scheme
	k8sClient, err := client.New(cfg, client.Options{Scheme: s})
	if err != nil {
		fmt.Printf("Error creating Kubernetes client: %v\n", err)
		return nil, err
	}
	return k8sClient, nil
}
