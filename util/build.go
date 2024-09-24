package util

import (
	"context"
	"fmt"

	buildv1 "github.com/forge-build/forge/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/cluster-api/util/annotations"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// BuildToInfrastructureMapFunc returns a handler.ToRequestsFunc that watches for
// Build events and returns reconciliation requests for an infrastructure provider object.
func BuildToInfrastructureMapFunc(ctx context.Context, gvk schema.GroupVersionKind, c client.Client, providerBuild client.Object) handler.MapFunc {
	log := ctrl.LoggerFrom(ctx)
	return func(ctx context.Context, o client.Object) []reconcile.Request {
		build, ok := o.(*buildv1.Build)
		if !ok {
			return nil
		}

		// Return early if the InfrastructureRef is nil.
		if build.Spec.InfrastructureRef == nil {
			return nil
		}
		gk := gvk.GroupKind()
		// Return early if the GroupKind doesn't match what we expect.
		infraGK := build.Spec.InfrastructureRef.GroupVersionKind().GroupKind()
		if gk != infraGK {
			return nil
		}
		providerBuild := providerBuild.DeepCopyObject().(client.Object)
		key := types.NamespacedName{Namespace: build.Namespace, Name: build.Spec.InfrastructureRef.Name}

		if err := c.Get(ctx, key, providerBuild); err != nil {
			log.V(4).Error(err, fmt.Sprintf("Failed to get %T", providerBuild))
			return nil
		}

		if annotations.IsExternallyManaged(providerBuild) {
			log.V(4).Info(fmt.Sprintf("%T is externally managed, skipping mapping", providerBuild))
			return nil
		}

		return []reconcile.Request{
			{
				NamespacedName: client.ObjectKey{
					Namespace: build.Namespace,
					Name:      build.Spec.InfrastructureRef.Name,
				},
			},
		}
	}
}
