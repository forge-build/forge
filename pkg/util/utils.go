package util

import (
	"context"

	buildv1 "github.com/forge-build/forge/pkg/api/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetOwnerBuild returns the Build object owning the current resource.
func GetOwnerBuild(ctx context.Context, c client.Client, obj metav1.ObjectMeta) (*buildv1.Build, error) {
	for _, ref := range obj.GetOwnerReferences() {
		if ref.Kind != "Build" {
			continue
		}
		gv, err := schema.ParseGroupVersion(ref.APIVersion)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		if gv.Group == buildv1.GroupVersion.Group {
			return GetBuildByName(ctx, c, obj.Namespace, ref.Name)
		}
	}
	return nil, nil
}

// GetBuildByName finds and return a Build object using the specified params.
func GetBuildByName(ctx context.Context, c client.Client, namespace, name string) (*buildv1.Build, error) {
	build := &buildv1.Build{}
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}

	if err := c.Get(ctx, key, build); err != nil {
		return nil, errors.Wrapf(err, "failed to get Build/%s", name)
	}

	return build, nil
}

func GetProvisionerByID(build *buildv1.Build, id string) (*buildv1.ProvisionerSpec, error) {
	for i := range build.Spec.Provisioners {
		if ptr.Deref(build.Spec.Provisioners[i].UUID, "") == id {
			return &build.Spec.Provisioners[i], nil
		}
	}
	return &buildv1.ProvisionerSpec{}, errors.Errorf("provisioner with ID %q not found in Build %q", id, build.Name)
}

// GetSecretFromSecretReference returns the secret data from the secret reference.
func GetSecretFromSecretReference(ctx context.Context, c client.Client, secretRef corev1.SecretReference) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	key := client.ObjectKey{
		Namespace: secretRef.Namespace,
		Name:      secretRef.Name,
	}
	if err := c.Get(ctx, key, secret); err != nil {
		return secret, errors.Wrapf(err, "failed to get Secret/%s", secretRef.Name)
	}

	return secret, nil
}
