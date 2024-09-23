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

package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	buildv1 "github.com/forge-build/forge/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//var _ = Describe("Build Controller", func() {
//	Context("When reconciling a resource", func() {
//		const resourceName = "test-resource"
//
//		ctx := context.Background()
//
//		typeNamespacedName := types.NamespacedName{
//			Name:      resourceName,
//			Namespace: "default", // TODO(user):Modify as needed
//		}
//		build := &buildv1.Build{}
//
//		BeforeEach(func() {
//			By("creating the custom resource for the Kind Build")
//			err := k8sClient.Get(ctx, typeNamespacedName, build)
//			if err != nil && errors.IsNotFound(err) {
//				resource := &buildv1.Build{
//					ObjectMeta: metav1.ObjectMeta{
//						Name:      resourceName,
//						Namespace: "default",
//					},
//					Spec: buildv1.BuildSpec{
//						InfrastructureRef: &corev1.ObjectReference{},
//					},
//					// TODO(user): Specify other spec details if needed.
//				}
//				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
//			}
//		})
//
//		AfterEach(func() {
//			// TODO(user): Cleanup logic after each test, like removing the resource instance.
//			resource := &buildv1.Build{}
//			err := k8sClient.Get(ctx, typeNamespacedName, resource)
//			Expect(err).NotTo(HaveOccurred())
//
//			By("Cleanup the specific resource instance Build")
//			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
//		})
//		It("should successfully reconcile the resource", func() {
//			By("Reconciling the created resource")
//			controllerReconciler := &BuildReconciler{
//				Client: k8sClient,
//			}
//
//			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
//				NamespacedName: typeNamespacedName,
//			})
//			Expect(err).NotTo(HaveOccurred())
//			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
//			// Example: If you expect a certain status condition after reconciliation, verify it here.
//		})
//	})
//})

var _ = Describe("BuildReconciler", func() {
	BeforeEach(func() {})
	AfterEach(func() {})

	Context("Reconcile a Build", func() {
		It("should not error and not requeue the request with insufficient set up", func() {
			ctx := context.Background()

			reconciler := &BuildReconciler{
				Client: k8sClient,
			}

			instance := &buildv1.Build{ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
				Spec: buildv1.BuildSpec{
					InfrastructureRef: &corev1.ObjectReference{
						APIVersion: "infrastructure.forge.build/v1alpha1",
						Kind:       "GCPBuild",
						Name:       "bar",
					},
				}}

			// Create the GCPCluster object and expect the Reconcile and Deployment to be created
			Expect(k8sClient.Create(ctx, instance)).To(Succeed())
			defer func() {
				err := k8sClient.Delete(ctx, instance)
				Expect(err).NotTo(HaveOccurred())
			}()

			result, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: client.ObjectKey{
					Namespace: instance.Namespace,
					Name:      instance.Name,
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeZero())
			Expect(result.Requeue).To(BeFalse())
		})
	})
})
