/*
Copyright 2020 The Kubernetes Authors.

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

// Package predicates implements predicate utilities.
package predicates

import (
	"fmt"

	buildv1 "github.com/forge-build/forge/api/v1alpha1"

	"github.com/go-logr/logr"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// ClusterCreateInfraReady returns a predicate that returns true for a create event when a cluster has Status.InfrastructureReady set as true
// it also returns true if the resource provided is not a Cluster to allow for use with controller-runtime NewControllerManagedBy.
func ClusterCreateInfraReady(logger logr.Logger) predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			log := logger.WithValues("predicate", "ClusterCreateInfraReady", "eventType", "create")

			c, ok := e.Object.(*clusterv1.Cluster)
			if !ok {
				log.V(4).Info("Expected Cluster", "type", fmt.Sprintf("%T", e.Object))
				return false
			}
			log = log.WithValues("Cluster", klog.KObj(c))

			// Only need to trigger a reconcile if the Cluster.Status.InfrastructureReady is true
			if c.Status.InfrastructureReady {
				log.V(6).Info("Cluster infrastructure is ready, allowing further processing")
				return true
			}

			log.V(4).Info("Cluster infrastructure is not ready, blocking further processing")
			return false
		},
		UpdateFunc:  func(event.UpdateEvent) bool { return false },
		DeleteFunc:  func(event.DeleteEvent) bool { return false },
		GenericFunc: func(event.GenericEvent) bool { return false },
	}
}

// ClusterCreateNotPaused returns a predicate that returns true for a create event when a cluster has Spec.Paused set as false
// it also returns true if the resource provided is not a Cluster to allow for use with controller-runtime NewControllerManagedBy.
func ClusterCreateNotPaused(logger logr.Logger) predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			log := logger.WithValues("predicate", "ClusterCreateNotPaused", "eventType", "create")

			c, ok := e.Object.(*clusterv1.Cluster)
			if !ok {
				log.V(4).Info("Expected Cluster", "type", fmt.Sprintf("%T", e.Object))
				return false
			}
			log = log.WithValues("Cluster", klog.KObj(c))

			// Only need to trigger a reconcile if the Cluster.Spec.Paused is false
			if !c.Spec.Paused {
				log.V(6).Info("Cluster is not paused, allowing further processing")
				return true
			}

			log.V(4).Info("Cluster is paused, blocking further processing")
			return false
		},
		UpdateFunc:  func(event.UpdateEvent) bool { return false },
		DeleteFunc:  func(event.DeleteEvent) bool { return false },
		GenericFunc: func(event.GenericEvent) bool { return false },
	}
}

// ClusterUpdateInfraReady returns a predicate that returns true for an update event when a cluster has Status.InfrastructureReady changed from false to true
// it also returns true if the resource provided is not a Cluster to allow for use with controller-runtime NewControllerManagedBy.
func ClusterUpdateInfraReady(logger logr.Logger) predicate.Funcs {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			log := logger.WithValues("predicate", "ClusterUpdateInfraReady", "eventType", "update")

			oldCluster, ok := e.ObjectOld.(*clusterv1.Cluster)
			if !ok {
				log.V(4).Info("Expected Cluster", "type", fmt.Sprintf("%T", e.ObjectOld))
				return false
			}
			log = log.WithValues("Cluster", klog.KObj(oldCluster))

			newCluster := e.ObjectNew.(*clusterv1.Cluster)

			if !oldCluster.Status.InfrastructureReady && newCluster.Status.InfrastructureReady {
				log.V(6).Info("Cluster infrastructure became ready, allowing further processing")
				return true
			}

			log.V(4).Info("Cluster infrastructure did not become ready, blocking further processing")
			return false
		},
		CreateFunc:  func(event.CreateEvent) bool { return false },
		DeleteFunc:  func(event.DeleteEvent) bool { return false },
		GenericFunc: func(event.GenericEvent) bool { return false },
	}
}

// BuildUpdateUnpaused returns a predicate that returns true for an update event when a build has Spec.Paused changed from true to false
// it also returns true if the resource provided is not a Build to allow for use with controller-runtime NewControllerManagedBy.
func BuildUpdateUnpaused(logger logr.Logger) predicate.Funcs {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			log := logger.WithValues("predicate", "ClusterUpdateUnpaused", "eventType", "update")

			oldCluster, ok := e.ObjectOld.(*buildv1.Build)
			if !ok {
				log.V(4).Info("Expected Build", "type", fmt.Sprintf("%T", e.ObjectOld))
				return false
			}
			log = log.WithValues("Build", klog.KObj(oldCluster))

			newCluster := e.ObjectNew.(*buildv1.Build)

			if oldCluster.Spec.Paused && !newCluster.Spec.Paused {
				log.V(4).Info("Cluster was unpaused, allowing further processing")
				return true
			}

			// This predicate always work in "or" with Paused predicates
			// so the logs are adjusted to not provide false negatives/verbosity al V<=5.
			log.V(6).Info("Cluster was not unpaused, blocking further processing")
			return false
		},
		CreateFunc:  func(event.CreateEvent) bool { return false },
		DeleteFunc:  func(event.DeleteEvent) bool { return false },
		GenericFunc: func(event.GenericEvent) bool { return false },
	}
}

// BuildUnpaused returns a Predicate that returns true on Build creation events where Build.Spec.Paused is false
// and Update events when Build.Spec.Paused transitions to false.
// This implements a common requirement for many cluster-api and provider controllers (such as Build Infrastructure
// controllers) to resume reconciliation when the Build is unpaused.
// Example use:
//
//	err := controller.Watch(
//	    source.Kind(cache, &buildv1.Cluster{}),
//	    handler.EnqueueRequestsFromMapFunc(buildToInfrastructureMapFunc),
//	    predicates.BuildUnpaused(r.Log),
//	)
func BuildUnpaused(logger logr.Logger) predicate.Funcs {
	log := logger.WithValues("predicate", "BuildUnpaused")

	// Use any to ensure we process either create or update events we care about
	return Any(log, ClusterCreateNotPaused(log), BuildUpdateUnpaused(log))
}

// ClusterUnpausedAndInfrastructureReady returns a Predicate that returns true on Cluster creation events where
// both Cluster.Spec.Paused is false and Cluster.Status.InfrastructureReady is true and Update events when
// either Cluster.Spec.Paused transitions to false or Cluster.Status.InfrastructureReady transitions to true.
// This implements a common requirement for some cluster-api and provider controllers (such as Machine Infrastructure
// controllers) to resume reconciliation when the Cluster is unpaused and when the infrastructure becomes ready.
// Example use:
//
//	err := controller.Watch(
//	    source.Kind(cache, &clusterv1.Cluster{}),
//	    handler.EnqueueRequestsFromMapFunc(clusterToMachines)
//	    predicates.ClusterUnpausedAndInfrastructureReady(r.Log),
//	)
func ClusterUnpausedAndInfrastructureReady(logger logr.Logger) predicate.Funcs {
	log := logger.WithValues("predicate", "ClusterUnpausedAndInfrastructureReady")

	// Only continue processing create events if both not paused and infrastructure is ready
	createPredicates := All(log, ClusterCreateNotPaused(log), ClusterCreateInfraReady(log))

	// Process update events if either Cluster is unpaused or infrastructure becomes ready
	updatePredicates := Any(log, BuildUpdateUnpaused(log), ClusterUpdateInfraReady(log))

	// Use any to ensure we process either create or update events we care about
	return Any(log, createPredicates, updatePredicates)
}
