package controller

import (
	"context"
	"fmt"
	"time"

	builderror "github.com/forge-build/forge/pkg/errors"

	"k8s.io/utils/ptr"

	buildv1 "github.com/forge-build/forge/api/v1alpha1"
	"github.com/forge-build/forge/provisioner/shell/job"
	"github.com/google/uuid"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	ShellProvisionerRepo = "ghcr.io/forge-build/forge-provisioner-shell"
	ShellProvisionerTag  = "latest"

	ForgeCoreNamespace = "forge-core"
)

func Reconcile(ctx context.Context, client client.Client, build *buildv1.Build, spec *buildv1.ProvisionerSpec) (_ ctrl.Result, err error) {

	// Create the Job
	if spec.UUID == nil {
		id := uuid.New()
		builder := job.NewShellJobBuilder().
			WithNamespace(ForgeCoreNamespace).
			WithBuildNamespace(build.Namespace).
			WithBuildName(build.Name).
			WithUUID(id.String()).
			// TODO get repo and tag from variables
			WithRepo("medchiheb/forge-shell-provisioner").
			WithTag("dev").
			WithBackOffLimit(ptr.Deref(spec.Retries, 1)).
			WithSSHCredentialsSecretName(build.Spec.Connector.Credentials.Name)

		if spec.Run != nil {
			builder.WithScriptToRun(*spec.Run)
		}
		if spec.RunConfigMapRef != nil {
			builder.WithScriptToRun(*spec.Run)
		}

		desired, err := builder.Build()
		if err != nil {
			return ctrl.Result{}, err
		}

		op, err := controllerutil.CreateOrPatch(ctx, client, desired, func() error {
			return nil
		})
		if err != nil {
			return ctrl.Result{}, err
		}

		spec.UUID = ptr.To(id.String())
		spec.Status = ptr.To(buildv1.ProvisionerStatusRunning)
		if op != controllerutil.OperationResultNone {
			// After job created we RequeueAfter 2 seconds.
			return ctrl.Result{
				RequeueAfter: 2 * time.Second,
			}, nil
		}
	}

	switch *spec.Status {
	case buildv1.ProvisionerStatusPending:
	case buildv1.ProvisionerStatusRunning:
		// RequeueAfter 2 seconds.
		return ctrl.Result{
			RequeueAfter: 2 * time.Second,
		}, nil
	case buildv1.ProvisionerStatusCompleted:
		// Requeue to check any other provisioner.
		return ctrl.Result{}, nil
	case buildv1.ProvisionerStatusFailed:
		// check if provisioner allowed to fail.
		if spec.AllowFail {
			return ctrl.Result{}, nil
		}
		// Fail the Build if provisioner failed.
		build.Status.FailureReason = ptr.To(builderror.ProvisionerFailedError)
		build.Status.FailureMessage = ptr.To(fmt.Sprintf("Provisioner %s failed with Reason %s and Message %s", *spec.UUID, *spec.FailureReason, *spec.FailureMessage))
		return ctrl.Result{}, nil
	default:
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}
