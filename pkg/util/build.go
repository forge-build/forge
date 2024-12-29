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

	buildv1 "github.com/forge-build/forge/pkg/api/v1alpha1"
	"github.com/forge-build/forge/pkg/kubernetes"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
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

		if kubernetes.IsExternallyManaged(providerBuild) {
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
