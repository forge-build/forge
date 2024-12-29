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

package util

import (
	"context"
	"fmt"

	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/cluster-api/util/record"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	buildv1 "github.com/forge-build/forge/pkg/api/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type SSHCredentials struct {
	Host       string
	Username   string
	Password   string
	PrivateKey string
	PublicKey  string
}

// EnsureCredentialsSecret ensures that the Build has a secret with the SSH credentials.
func EnsureCredentialsSecret(ctx context.Context, client client.Client, build *buildv1.Build, creds SSHCredentials, provider string) error {
	patchHelper, err := patch.NewHelper(build, client)
	if err != nil {
		return err
	}

	name := fmt.Sprintf("%s-ssh-credentials", build.Name)
	credentials := &corev1.Secret{
		Type: buildv1.BuildSecretType,
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: build.Namespace,
			Labels: map[string]string{
				buildv1.BuildNameLabel: build.Name,
			},
			Annotations: map[string]string{
				buildv1.ManagedByAnnotation: "forge",
				buildv1.ProviderNameLabel:   provider,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					Name:       build.Name,
					UID:        build.GetUID(),
					APIVersion: build.APIVersion,
					Kind:       build.Kind,
				},
			},
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, client, credentials, func() error {
		// Ensure Labels and Annotations are set
		if credentials.Labels == nil {
			credentials.Labels = map[string]string{}
		}
		credentials.Labels[buildv1.BuildNameLabel] = build.Name

		if credentials.Annotations == nil {
			credentials.Annotations = map[string]string{}
		}
		credentials.Annotations[buildv1.ManagedByAnnotation] = "forge"
		credentials.Annotations[buildv1.ProviderNameLabel] = provider

		// Update the data fields
		if credentials.StringData == nil {
			credentials.StringData = map[string]string{}
		}
		if creds.Username != "" {
			credentials.StringData["username"] = creds.Username
		}
		if creds.Host != "" {
			credentials.StringData["host"] = creds.Host
		}
		if creds.Password != "" {
			credentials.StringData["password"] = creds.Password
		}
		if creds.PrivateKey != "" {
			credentials.StringData["privateKey"] = creds.PrivateKey
		}
		if creds.PublicKey != "" {
			credentials.StringData["publicKey"] = creds.PublicKey
		}

		return nil
	})
	if err != nil {
		return errors.Wrap(err, "unable to create ssh credentials secret")
	}

	if op != controllerutil.OperationResultNone {
		record.Eventf(build, "SSHCredentials", "Build Got SSH Credentials Secret %s", name)
	}

	// patch Build to include the credential secret.
	// TODO: make this as a contract,
	// no need for infrabuilds to set the secret name, they should do it, in their spec.
	// so the Build will read it.
	build.Spec.Connector.Credentials = &corev1.LocalObjectReference{Name: name}

	err = patchHelper.Patch(ctx, build)
	if err != nil {
		return errors.Wrap(err, "unable to patch Build")
	}

	return nil
}
