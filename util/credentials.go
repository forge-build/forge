package util

import (
	"context"
	"fmt"

	"sigs.k8s.io/cluster-api/util/record"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	buildv1 "github.com/forge-build/forge/api/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type SSHCredentials struct {
	Host       string
	Username   string
	Password   string
	PrivateKey string
	PublicKey  string
}

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
		StringData: map[string]string{
			"host":     creds.Host,
			"username": creds.Username,
		},
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

	op, err := controllerutil.CreateOrUpdate(ctx, client, credentials, func() error { return nil })
	if err != nil {
		return errors.Wrap(err, "unable to create ssh credentials secret")
	}

	if op != controllerutil.OperationResultNone {
		record.Eventf(build, "SSHCredentials", "Build Got SSH Credentials Secret %s", name)
	}

	// patch Build to include the credential secret.
	build.Spec.Connector.Credentials = &corev1.LocalObjectReference{Name: name}

	err = patchHelper.Patch(ctx, build)
	if err != nil {
		return errors.Wrap(err, "unable to patch Build")
	}

	return nil
}
