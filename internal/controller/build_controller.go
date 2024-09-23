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
	"fmt"
	"github.com/forge-build/forge/utils"
	utilconversion "github.com/forge-build/forge/utils/conversion"
	"github.com/forge-build/forge/utils/predicates"
	"path"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/cluster-api/util"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	buildv1 "github.com/forge-build/forge/api/v1alpha1"
	"github.com/forge-build/forge/internal/external"
	forgeerrors "github.com/forge-build/forge/pkg/errors"
)

const (
	// deleteRequeueAfter is how long to wait before checking again to see if the cluster still has children during
	// deletion.
	deleteRequeueAfter = 5 * time.Second
)

// BuildReconciler reconciles a Build object
type BuildReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Logger logr.Logger

	// WatchFilterValue is the label value used to filter events prior to reconciliation.
	WatchFilterValue string

	recorder        record.EventRecorder
	externalTracker external.ObjectTracker
}

// SetupWithManager sets up the controller with the Manager.
func (r *BuildReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	c, err := ctrl.NewControllerManagedBy(mgr).
		For(&buildv1.Build{}).
		WithOptions(options).
		WithEventFilter(predicates.ResourceNotPausedAndHasFilterLabel(ctrl.LoggerFrom(ctx), r.WatchFilterValue)).
		Build(r)

	if err != nil {
		return errors.Wrap(err, "failed setting up with a controller manager")
	}

	r.recorder = mgr.GetEventRecorderFor("cluster-controller")
	r.externalTracker = external.ObjectTracker{
		Controller: c,
		Cache:      mgr.GetCache(),
	}
	return nil
}

//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;patch;update
//+kubebuilder:rbac:groups=infrastructure.forge.build;provisioner.forge.build,resources=*,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=forge.build,resources=builds,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=forge.build,resources=builds/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=forge.build,resources=builds/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *BuildReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	r.Logger = ctrl.LoggerFrom(ctx)

	// Fetch the Cluster instance.
	build := &buildv1.Build{}
	if err := r.Client.Get(ctx, req.NamespacedName, build); err != nil {
		if apierrors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return ctrl.Result{}, nil
		}

		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	// Return early if the object or Cluster is paused.
	if utils.IsPaused(build, build) {
		r.Logger.Info("Reconciliation is paused for this object")
		return ctrl.Result{}, nil
	}

	// Initialize the patch helper.
	patchHelper, err := patch.NewHelper(build, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}

	defer func() {
		// Always reconcile the Status.Phase field.
		r.reconcilePhase(ctx, build)

		// Always attempt to Patch the Cluster object and status after each reconciliation.
		// Patch ObservedGeneration only if the reconciliation is completed successfully
		patchOpts := []patch.Option{}
		if reterr == nil {
			patchOpts = append(patchOpts, patch.WithStatusObservedGeneration{})
		}
		if err := patchBuild(ctx, patchHelper, build, patchOpts...); err != nil {
			reterr = kerrors.NewAggregate([]error{reterr, err})
		}
	}()

	// Handle deletion reconciliation loop.
	if !build.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, build)
	}

	// Add finalizer first if not set to avoid the race condition between init and delete.
	// Note: Finalizers in general can only be added when the deletionTimestamp is not set.
	if !controllerutil.ContainsFinalizer(build, buildv1.BuildFinalizer) {
		controllerutil.AddFinalizer(build, buildv1.BuildFinalizer)
		return ctrl.Result{}, nil
	}

	// Handle normal reconciliation loop.
	return r.reconcile(ctx, build)
}

func patchBuild(ctx context.Context, patchHelper *patch.Helper, build *buildv1.Build, options ...patch.Option) error {
	// Always update the readyCondition by summarizing the state of other conditions.
	conditions.SetSummary(build,
		conditions.WithConditions(
			buildv1.ProvisionersReadyCondition,
			buildv1.InfrastructureReadyCondition,
		),
	)

	// Patch the object, ignoring conflicts on the conditions owned by this controller.
	// Also, if requested, we are adding additional options like e.g. Patch ObservedGeneration when issuing the
	// patch at the end of the reconcile loop.
	options = append(options,
		patch.WithOwnedConditions{Conditions: []clusterv1.ConditionType{
			buildv1.ReadyCondition,
			buildv1.ProvisionersReadyCondition,
			buildv1.InfrastructureReadyCondition,
		}},
	)
	return patchHelper.Patch(ctx, build, options...)
}

// reconcile handles cluster reconciliation.
func (r *BuildReconciler) reconcile(ctx context.Context, build *buildv1.Build) (ctrl.Result, error) {
	phases := []func(context.Context, *buildv1.Build) (ctrl.Result, error){
		r.reconcileInfrastructure,
		r.reconcileConnection,
		r.reconcileProvisioners,
		r.reconcileImageProvided,
	}

	res := ctrl.Result{}
	var errs []error
	for _, phase := range phases {
		// Call the inner reconciliation methods.
		phaseResult, err := phase(ctx, build)
		if err != nil {
			errs = append(errs, err)
		}
		if len(errs) > 0 {
			continue
		}
		res = util.LowestNonZeroResult(res, phaseResult)
	}
	return res, kerrors.NewAggregate(errs)
}

// reconcileDelete handles cluster deletion.
func (r *BuildReconciler) reconcileDelete(ctx context.Context, build *buildv1.Build) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	descendants, err := r.listDescendants(ctx, build)
	if err != nil {
		log.Error(err, "Failed to list descendants")
		return reconcile.Result{}, err
	}

	children, err := descendants.filterOwnedDescendants(build)
	if err != nil {
		log.Error(err, "Failed to extract direct descendants")
		return reconcile.Result{}, err
	}

	if len(children) > 0 {
		log.Info("Build still has children - deleting them first", "count", len(children))

		var errs []error

		for _, child := range children {
			if !child.GetDeletionTimestamp().IsZero() {
				// Don't handle deleted child
				continue
			}

			gvk, err := apiutil.GVKForObject(child, r.Client.Scheme())
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "error getting gvk for child object"))
				continue
			}

			log := log.WithValues(gvk.Kind, klog.KObj(child))
			log.Info("Deleting child object")
			if err := r.Client.Delete(ctx, child); err != nil {
				err = errors.Wrapf(err, "error deleting cluster %s/%s: failed to delete %s %s", build.Namespace, build.Name, gvk, child.GetName())
				log.Error(err, "Error deleting resource")
				errs = append(errs, err)
			}
		}

		if len(errs) > 0 {
			return ctrl.Result{}, kerrors.NewAggregate(errs)
		}
	}

	if descendantCount := descendants.length(); descendantCount > 0 {
		indirect := descendantCount - len(children)
		log.Info("Build still has descendants - need to requeue", "descendants", descendants.descendantNames(), "indirect descendants count", indirect)
		// Requeue so we can check the next time to see if there are still any descendants left.
		return ctrl.Result{RequeueAfter: deleteRequeueAfter}, nil
	}

	if build.Spec.InfrastructureRef != nil {
		obj, err := external.Get(ctx, r.Client, build.Spec.InfrastructureRef, build.Namespace)
		switch {
		case apierrors.IsNotFound(errors.Cause(err)):
			// All good - the InfraBuild resource has been deleted
			conditions.MarkFalse(build, buildv1.InfrastructureReadyCondition, buildv1.DeletedReason, buildv1.ConditionSeverityInfo, "")
		case err != nil:
			return reconcile.Result{}, errors.Wrapf(err, "failed to get %s %q for Build %s/%s",
				path.Join(build.Spec.InfrastructureRef.APIVersion, build.Spec.InfrastructureRef.Kind),
				build.Spec.InfrastructureRef.Name, build.Namespace, build.Name)
		default:
			// Report a summary of current status of the InfraBuild object defined for this build.
			conditions.SetMirror(build, buildv1.InfrastructureReadyCondition,
				conditions.UnstructuredGetter(obj),
				conditions.WithFallbackValue(false, clusterv1.DeletingReason, clusterv1.ConditionSeverityInfo, ""),
			)

			// Issue a deletion request for the InfraBuild object.
			// Once it's been deleted, the build will get processed again.
			if err := r.Client.Delete(ctx, obj); err != nil {
				return ctrl.Result{}, errors.Wrapf(err,
					"failed to delete %v %q for Build %q in namespace %q",
					obj.GroupVersionKind(), obj.GetName(), build.Name, build.Namespace)
			}

			// Return here so we don't remove the finalizer yet.
			log.Info("Build still has descendants - need to requeue", "InfrastructureRef", build.Spec.InfrastructureRef.Name)
			return ctrl.Result{}, nil
		}
	}

	controllerutil.RemoveFinalizer(build, buildv1.BuildFinalizer)
	r.recorder.Eventf(build, corev1.EventTypeNormal, "Deleted", "Build %s has been deleted", build.Name)
	return ctrl.Result{}, nil
}

// reconcileInfrastructure reconciles the Spec.InfrastructureRef object on a Build.
func (r *BuildReconciler) reconcileInfrastructure(ctx context.Context, build *buildv1.Build) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	if build.Spec.InfrastructureRef == nil {
		return ctrl.Result{}, nil
	}

	// Call generic external reconciler.
	infraReconcileResult, err := r.reconcileExternal(ctx, build, build.Spec.InfrastructureRef)
	if err != nil {
		return ctrl.Result{}, err
	}
	// Return early if we need to requeue.
	if infraReconcileResult.RequeueAfter > 0 {
		return ctrl.Result{RequeueAfter: infraReconcileResult.RequeueAfter}, nil
	}
	// If the external object is paused, return without any further processing.
	if infraReconcileResult.Paused {
		return ctrl.Result{}, nil
	}
	infraConfig := infraReconcileResult.Result

	// There's no need to go any further if the Build is marked for deletion.
	if !infraConfig.GetDeletionTimestamp().IsZero() {
		return ctrl.Result{}, nil
	}

	// Determine if the infrastructure provider is ready.
	preReconcileInfrastructureReady := build.Status.InfrastructureReady
	ready, err := external.IsReady(infraConfig)
	if err != nil {
		return ctrl.Result{}, err
	}
	build.Status.InfrastructureReady = ready
	// Only record the event if the status has changed
	if preReconcileInfrastructureReady != build.Status.InfrastructureReady {
		r.recorder.Eventf(build, corev1.EventTypeNormal, "InfrastructureReady", "Build %s InfrastructureReady is now %t", build.Name, build.Status.InfrastructureReady)
	}

	// Report a summary of current status of the infrastructure object defined for this cluster.
	conditions.SetMirror(build, buildv1.InfrastructureReadyCondition,
		conditions.UnstructuredGetter(infraConfig),
		conditions.WithFallbackValue(ready, buildv1.WaitingForInfrastructureFallbackReason, buildv1.ConditionSeverityInfo, ""),
	)

	if !ready {
		log.V(3).Info("Infrastructure provider is not ready yet")
		return ctrl.Result{}, nil
	}

	// Get and parse Status.FailureDomains from the infrastructure provider.
	failureDomains := buildv1.FailureDomains{}
	if err := utils.UnstructuredUnmarshalField(infraConfig, &failureDomains, "status", "failureDomains"); err != nil && err != utils.ErrUnstructuredFieldNotFound {
		return ctrl.Result{}, errors.Wrapf(err, "failed to retrieve Status.FailureDomains from infrastructure provider for Build %q in namespace %q",
			build.Name, build.Namespace)
	}
	build.Status.FailureDomains = failureDomains

	return ctrl.Result{}, nil
}

// reconcileExternal handles generic unstructured objects referenced by a Cluster.
func (r *BuildReconciler) reconcileExternal(ctx context.Context, build *buildv1.Build, ref *corev1.ObjectReference) (external.ReconcileOutput, error) {
	log := ctrl.LoggerFrom(ctx)

	if err := utilconversion.UpdateReferenceAPIContract(ctx, r.Client, ref); err != nil {
		return external.ReconcileOutput{}, err
	}

	obj, err := external.Get(ctx, r.Client, ref, build.Namespace)
	if err != nil {
		if apierrors.IsNotFound(errors.Cause(err)) {
			log.Info("Could not find external object for build, requeuing", "refGroupVersionKind", ref.GroupVersionKind(), "refName", ref.Name)
			return external.ReconcileOutput{RequeueAfter: 30 * time.Second}, nil
		}
		return external.ReconcileOutput{}, err
	}

	// Ensure we add a watcher to the external object.
	if err := r.externalTracker.Watch(log, obj, handler.EnqueueRequestForOwner(r.Client.Scheme(), r.Client.RESTMapper(), &buildv1.Build{})); err != nil {
		return external.ReconcileOutput{}, err
	}

	// if external ref is paused, return error.
	if utils.IsPaused(build, obj) {
		log.V(3).Info("External object referenced is paused")
		return external.ReconcileOutput{Paused: true}, nil
	}

	// Initialize the patch helper.
	patchHelper, err := patch.NewHelper(obj, r.Client)
	if err != nil {
		return external.ReconcileOutput{}, err
	}

	// Set external object ControllerReference to the Cluster.
	if err := controllerutil.SetControllerReference(build, obj, r.Client.Scheme()); err != nil {
		return external.ReconcileOutput{}, err
	}

	// Set the Build label.
	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[buildv1.BuildNameLabel] = build.Name
	obj.SetLabels(labels)

	// Always attempt to Patch the external object.
	if err := patchHelper.Patch(ctx, obj); err != nil {
		return external.ReconcileOutput{}, err
	}

	// Set failure reason and message, if any.
	failureReason, failureMessage, err := external.FailuresFrom(obj)
	if err != nil {
		return external.ReconcileOutput{}, err
	}
	if failureReason != "" {
		clusterStatusError := forgeerrors.BuildStatusError(failureReason)
		build.Status.FailureReason = &clusterStatusError
	}
	if failureMessage != "" {
		build.Status.FailureMessage = ptr.To(
			fmt.Sprintf("Failure detected from referenced resource %v with name %q: %s",
				obj.GroupVersionKind(), obj.GetName(), failureMessage),
		)
	}

	return external.ReconcileOutput{Result: obj}, nil
}

// reconcileImageProvided reconciles the InfraBuild to process the exportation of the image.
func (r *BuildReconciler) reconcileImageProvided(ctx context.Context, build *buildv1.Build) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Skip checking if the Provisioners not ready.
	if !build.Status.ProvisionersReady {
		log.V(4).Info("Skipping reconcileImageProvided because Provisioners not ready yet")
		return ctrl.Result{}, nil
	}

	if build.Status.Ready && conditions.IsTrue(build, buildv1.ReadyCondition) {
		log.V(4).Info("Skipping reconcileImageProvided because Build already provided")
		return ctrl.Result{}, nil
	}

	log.V(4).Info("Checking for image exportation")
	// TODO, Mark the InfraBuild to export the image.

	conditions.MarkTrue(build, buildv1.BuildInitializedCondition)
	return ctrl.Result{}, nil
}

// reconcileConnection reconciles the connection to the underlying infrastructure machine.
func (r *BuildReconciler) reconcileConnection(ctx context.Context, build *buildv1.Build) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Skip checking if the Infrastructure not ready.
	if !build.Status.InfrastructureReady {
		log.V(4).Info("Skipping reconcileConnection because Infrastructure not ready yet")
		return ctrl.Result{}, nil
	}

	if build.Status.Connected {
		log.V(4).Info("Skipping reconcileConnection because it is already connected")
		return ctrl.Result{}, nil
	}

	log.V(4).Info("Checking for connection to infrastructure machine")
	conditions.MarkFalse(build, buildv1.BuildInitializedCondition, buildv1.WaitingForConnectionReason, buildv1.ConditionSeverityInfo, "")
	// TODO, Try to connect to the infrastructure machine with spec.connector.

	return ctrl.Result{}, nil
}

// reconcileProvisioners reconciles the provisioners for the Build.
func (r *BuildReconciler) reconcileProvisioners(ctx context.Context, build *buildv1.Build) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Skip checking if the Infrastructure not ready.
	if !build.Status.Connected {
		log.V(4).Info("Skipping reconcileProvisioners because the infrastructure machine is not connected yet")
		return ctrl.Result{}, nil
	}

	if build.Status.ProvisionersReady {
		log.V(4).Info("Skipping reconcileProvisioners because provisioners are ready")
		return ctrl.Result{}, nil
	}

	log.V(4).Info("Checking for provisioners")
	conditions.MarkFalse(build, buildv1.ProvisionersReadyCondition, buildv1.WaitingForProvisionersReason, buildv1.ConditionSeverityInfo, "")
	// TODO, Mark the provisioners to run.

	return ctrl.Result{}, nil
}

type buildDescendants struct {
	infraBuild   unstructured.UnstructuredList
	provisioners unstructured.UnstructuredList
}

// length returns the number of descendants.
func (c *buildDescendants) length() int {
	return len(c.infraBuild.Items) +
		len(c.provisioners.Items)
}

// listDescendants returns a list of all InfraBuilds, and Provisioners for the Build.
func (r *BuildReconciler) listDescendants(ctx context.Context, build *buildv1.Build) (buildDescendants, error) {
	var descendants buildDescendants

	listOptions := []client.ListOption{
		client.InNamespace(build.Namespace),
		client.MatchingLabels(map[string]string{buildv1.BuildNameLabel: build.Name}),
	}

	// retrieve InfraBuild
	infraBuildGVK := build.Spec.InfrastructureRef.GroupVersionKind()
	descendants.infraBuild.SetGroupVersionKind(infraBuildGVK)
	err := r.List(ctx, &descendants.infraBuild, listOptions...)
	if err != nil {
		return descendants, errors.Wrapf(err, "failed to list objects with kind '%s'", infraBuildGVK.Kind)
	}

	// retrieve Provisioners
	descendants.provisioners = unstructured.UnstructuredList{}
	for _, p := range build.Spec.Provisioners {
		if p.Type == buildv1.ProvisionerTypeExternal {
			var provisionersList unstructured.UnstructuredList
			provisionerGVK := p.Ref.GroupVersionKind()
			provisionersList.SetGroupVersionKind(provisionerGVK)
			err = r.List(ctx, &provisionersList, listOptions...)
			if err != nil {
				return descendants, errors.Wrapf(err, "failed to list objects with kind '%s'", provisionerGVK.Kind)
			}
			descendants.provisioners.Items = append(descendants.provisioners.Items, provisionersList.Items...)
		}
	}

	return descendants, nil
}

// filterOwnedDescendants returns an array of runtime.Objects containing only those descendants that have the build
// as an owner reference, with infrabuild sorted last.
func (c buildDescendants) filterOwnedDescendants(build *buildv1.Build) ([]client.Object, error) {
	var ownedDescendants []client.Object
	eachFunc := func(o runtime.Object) error {
		obj := o.(client.Object)
		acc, err := meta.Accessor(obj)
		if err != nil {
			return nil //nolint:nilerr // We don't want to exit the EachListItem loop, just continue
		}

		if util.IsOwnedByObject(acc, build) {
			ownedDescendants = append(ownedDescendants, obj)
		}

		return nil
	}

	lists := []client.ObjectList{
		&c.provisioners,
		&c.infraBuild,
	}

	for _, list := range lists {
		if err := meta.EachListItem(list, eachFunc); err != nil {
			return nil, errors.Wrapf(err, "error finding owned descendants of build %s/%s", build.Namespace, build.Name)
		}
	}

	return ownedDescendants, nil
}

func (c *buildDescendants) descendantNames() string {
	descendants := make([]string, 0)
	infraBuildNames := make([]string, len(c.infraBuild.Items))
	for i, b := range c.infraBuild.Items {
		infraBuildNames[i] = b.GetName()
	}
	if len(infraBuildNames) > 0 {
		descendants = append(descendants, "InfraBuild: "+strings.Join(infraBuildNames, ","))
	}
	provisionersNames := make([]string, len(c.provisioners.Items))
	for i, p := range c.provisioners.Items {
		provisionersNames[i] = p.GetName()
	}
	if len(provisionersNames) > 0 {
		descendants = append(descendants, "Provisioners: "+strings.Join(provisionersNames, ","))
	}

	return strings.Join(descendants, ";")
}
