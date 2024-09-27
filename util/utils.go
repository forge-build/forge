package util

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/gobuffalo/flect"
	"github.com/pkg/errors"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sversion "k8s.io/apimachinery/pkg/version"
	"sigs.k8s.io/controller-runtime/pkg/client"

	buildv1 "github.com/forge-build/forge/api/v1alpha1"
)

var (
	// rnd = rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

	// ErrNoCluster is returned when the cluster
	// label could not be found on the object passed in.
	ErrNoCluster = fmt.Errorf("no %q label present", buildv1.BuildNameLabel)

	// ErrUnstructuredFieldNotFound determines that a field
	// in an unstructured object could not be found.
	ErrUnstructuredFieldNotFound = fmt.Errorf("field not found")
)

// IsNil returns an error if the passed interface is equal to nil or if it has an interface value of nil.
func IsNil(i interface{}) bool {
	if i == nil {
		return true
	}
	switch reflect.TypeOf(i).Kind() {
	case reflect.Ptr, reflect.Map, reflect.Chan, reflect.Slice, reflect.Interface, reflect.UnsafePointer, reflect.Func:
		return reflect.ValueOf(i).IsValid() && reflect.ValueOf(i).IsNil()
	}
	return false
}

// IsPaused returns true if the Cluster is paused or the object has the `paused` annotation.
func IsPaused(build *buildv1.Build, o metav1.Object) bool {
	if build.Spec.Paused {
		return true
	}
	return HasPaused(o)
}

// HasPaused returns true if the object has the `paused` annotation.
func HasPaused(o metav1.Object) bool {
	return hasAnnotation(o, buildv1.PausedAnnotation)
}

// hasAnnotation returns true if the object has the specified annotation.
func hasAnnotation(o metav1.Object, annotation string) bool {
	annotations := o.GetAnnotations()
	if annotations == nil {
		return false
	}
	_, ok := annotations[annotation]
	return ok
}

// HasWatchLabel returns true if the object has a label with the WatchLabel key matching the given value.
func HasWatchLabel(o metav1.Object, labelValue string) bool {
	val, ok := o.GetLabels()[buildv1.WatchLabel]
	if !ok {
		return false
	}
	return val == labelValue
}

// UnstructuredUnmarshalField is a wrapper around json and unstructured objects to decode and copy a specific field
// value into an object.
func UnstructuredUnmarshalField(obj *unstructured.Unstructured, v interface{}, fields ...string) error {
	if obj == nil || obj.Object == nil {
		return errors.Errorf("failed to unmarshal unstructured object: object is nil")
	}

	value, found, err := unstructured.NestedFieldNoCopy(obj.Object, fields...)
	if err != nil {
		return errors.Wrapf(err, "failed to retrieve field %q from %q", strings.Join(fields, "."), obj.GroupVersionKind())
	}
	if !found || value == nil {
		return ErrUnstructuredFieldNotFound
	}
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return errors.Wrapf(err, "failed to json-encode field %q value from %q", strings.Join(fields, "."), obj.GroupVersionKind())
	}
	if err := json.Unmarshal(valueBytes, v); err != nil {
		return errors.Wrapf(err, "failed to json-decode field %q value from %q", strings.Join(fields, "."), obj.GroupVersionKind())
	}
	return nil
}

// GetGVKMetadata retrieves a CustomResourceDefinition metadata from the API server using partial object metadata.
//
// This function is greatly more efficient than GetCRDWithContract and should be preferred in most cases.
func GetGVKMetadata(ctx context.Context, c client.Client, gvk schema.GroupVersionKind) (*metav1.PartialObjectMetadata, error) {
	meta := &metav1.PartialObjectMetadata{}
	meta.SetName(CalculateCRDName(gvk.Group, gvk.Kind))
	meta.SetGroupVersionKind(apiextensionsv1.SchemeGroupVersion.WithKind("CustomResourceDefinition"))
	if err := c.Get(ctx, client.ObjectKeyFromObject(meta), meta); err != nil {
		return meta, errors.Wrap(err, "failed to retrieve metadata from GVK resource")
	}
	return meta, nil
}

// CalculateCRDName generates a CRD name based on group and kind according to
// the naming conventions in the contract.
func CalculateCRDName(group, kind string) string {
	return fmt.Sprintf("%s.%s", flect.Pluralize(strings.ToLower(kind)), group)
}

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

// KubeAwareAPIVersions is a sortable slice of kube-like version strings.
//
// Kube-like version strings are starting with a v, followed by a major version,
// optional "alpha" or "beta" strings followed by a minor version (e.g. v1, v2beta1).
// Versions will be sorted based on GA/alpha/beta first and then major and minor
// versions. e.g. v2, v1, v1beta2, v1beta1, v1alpha1.
type KubeAwareAPIVersions []string

func (k KubeAwareAPIVersions) Len() int      { return len(k) }
func (k KubeAwareAPIVersions) Swap(i, j int) { k[i], k[j] = k[j], k[i] }
func (k KubeAwareAPIVersions) Less(i, j int) bool {
	return k8sversion.CompareKubeAwareVersionStrings(k[i], k[j]) < 0
}
