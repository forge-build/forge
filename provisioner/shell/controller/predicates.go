package controller

import (
	buildv1 "github.com/forge-build/forge/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// ManagedByForgeProvisionerShell is a predicate.Predicate that returns true if the
// specified client.Object is managed by Forge.
var ManagedByForgeProvisionerShell = predicate.NewPredicateFuncs(func(obj client.Object) bool {
	if managedBy, ok := obj.GetLabels()[buildv1.ManagedByLabel]; ok {
		return managedBy == ForgeProvisionerShellName
	}
	return false
})

// IsBeingTerminated is a predicate.Predicate that returns true if the specified
// client.Object is being terminated, i.e. its DeletionTimestamp property is set to non nil value.
var IsBeingTerminated = predicate.NewPredicateFuncs(func(obj client.Object) bool {
	return obj.GetDeletionTimestamp() != nil
})

// JobHasAnyCondition is a predicate.Predicate that returns true if the
// specified client.Object is a v1.Job with any v1.JobConditionType.
var JobHasAnyCondition = predicate.NewPredicateFuncs(func(obj client.Object) bool {
	if job, ok := obj.(*batchv1.Job); ok {
		return len(job.Status.Conditions) > 0
	}
	return false
})

// InNamespace is a predicate.Predicate that returns true if the
// specified client.Object is in the desired namespace.
var InNamespace = func(namespace string) predicate.Predicate {
	return predicate.NewPredicateFuncs(func(obj client.Object) bool {
		return namespace == obj.GetNamespace()
	})
}

var HasBuildNameLabel = predicate.NewPredicateFuncs(func(obj client.Object) bool {
	if _, ok := obj.GetLabels()[buildv1.BuildNameLabel]; ok {
		return true
	}
	return false
})

var HasProvisionerIDLabel = predicate.NewPredicateFuncs(func(obj client.Object) bool {
	if _, ok := obj.GetLabels()[buildv1.ProvisionerIDLabel]; ok {
		return true
	}
	return false
})
