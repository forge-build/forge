package predicates

import (
	"github.com/forge-build/forge/utils"
	"strings"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// ResourceNotPausedAndHasFilterLabel returns a predicate that returns true only if the
// ResourceNotPaused and ResourceHasFilterLabel predicates return true.
func ResourceNotPausedAndHasFilterLabel(logger logr.Logger, labelValue string) predicate.Funcs {
	return All(logger, ResourceNotPaused(logger), ResourceHasFilterLabel(logger, labelValue))
}

// All returns a predicate that returns true only if all given predicates return true.
func All(logger logr.Logger, predicates ...predicate.Funcs) predicate.Funcs {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			log := logger.WithValues("predicateAggregation", "All")
			for _, p := range predicates {
				if !p.UpdateFunc(e) {
					log.V(6).Info("One of the provided predicates returned false, blocking further processing")
					return false
				}
			}
			log.V(6).Info("All provided predicates returned true, allowing further processing")
			return true
		},
		CreateFunc: func(e event.CreateEvent) bool {
			log := logger.WithValues("predicateAggregation", "All")
			for _, p := range predicates {
				if !p.CreateFunc(e) {
					log.V(6).Info("One of the provided predicates returned false, blocking further processing")
					return false
				}
			}
			log.V(6).Info("All provided predicates returned true, allowing further processing")
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			log := logger.WithValues("predicateAggregation", "All")
			for _, p := range predicates {
				if !p.DeleteFunc(e) {
					log.V(6).Info("One of the provided predicates returned false, blocking further processing")
					return false
				}
			}
			log.V(6).Info("All provided predicates returned true, allowing further processing")
			return true
		},
		GenericFunc: func(e event.GenericEvent) bool {
			log := logger.WithValues("predicateAggregation", "All")
			for _, p := range predicates {
				if !p.GenericFunc(e) {
					log.V(6).Info("One of the provided predicates returned false, blocking further processing")
					return false
				}
			}
			log.V(6).Info("All provided predicates returned true, allowing further processing")
			return true
		},
	}
}

// ResourceHasFilterLabel returns a predicate that returns true only if the provided resource contains
// a label with the WatchLabel key and the configured label value exactly.
func ResourceHasFilterLabel(logger logr.Logger, labelValue string) predicate.Funcs {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return processIfLabelMatch(logger.WithValues("predicate", "ResourceHasFilterLabel", "eventType", "update"), e.ObjectNew, labelValue)
		},
		CreateFunc: func(e event.CreateEvent) bool {
			return processIfLabelMatch(logger.WithValues("predicate", "ResourceHasFilterLabel", "eventType", "create"), e.Object, labelValue)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return processIfLabelMatch(logger.WithValues("predicate", "ResourceHasFilterLabel", "eventType", "delete"), e.Object, labelValue)
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return processIfLabelMatch(logger.WithValues("predicate", "ResourceHasFilterLabel", "eventType", "generic"), e.Object, labelValue)
		},
	}
}

func ResourceNotPaused(logger logr.Logger) predicate.Funcs {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return processIfNotPaused(logger.WithValues("predicate", "ResourceNotPaused", "eventType", "update"), e.ObjectNew)
		},
		CreateFunc: func(e event.CreateEvent) bool {
			return processIfNotPaused(logger.WithValues("predicate", "ResourceNotPaused", "eventType", "create"), e.Object)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return processIfNotPaused(logger.WithValues("predicate", "ResourceNotPaused", "eventType", "delete"), e.Object)
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return processIfNotPaused(logger.WithValues("predicate", "ResourceNotPaused", "eventType", "generic"), e.Object)
		},
	}
}

func processIfNotPaused(logger logr.Logger, obj client.Object) bool {
	kind := strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind)
	log := logger.WithValues("namespace", obj.GetNamespace(), kind, obj.GetName())
	if utils.HasPaused(obj) {
		log.V(4).Info("Resource is paused, will not attempt to map resource")
		return false
	}
	log.V(6).Info("Resource is not paused, will attempt to map resource")
	return true
}

func processIfLabelMatch(logger logr.Logger, obj client.Object, labelValue string) bool {
	// Return early if no labelValue was set.
	if labelValue == "" {
		return true
	}

	kind := strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind)
	log := logger.WithValues("namespace", obj.GetNamespace(), kind, obj.GetName())
	if utils.HasWatchLabel(obj, labelValue) {
		log.V(6).Info("Resource matches label, will attempt to map resource")
		return true
	}
	log.V(4).Info("Resource does not match label, will not attempt to map resource")
	return false
}
