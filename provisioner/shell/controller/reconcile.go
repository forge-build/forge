package controller

import (
	"context"
	"time"

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
			WithRepo(ShellProvisionerRepo).
			WithTag(ShellProvisionerTag).
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

		op, err := controllerutil.CreateOrUpdate(ctx, client, desired, func() error {
			return nil
		})
		if err != nil {
			return ctrl.Result{}, err
		}

		spec.UUID = ptr.To(id.String())
		if op != controllerutil.OperationResultNone {
			// After job created we RequeueAfter 2 seconds.
			return ctrl.Result{
				RequeueAfter: 2 * time.Second,
			}, nil
		}
	}

	// Watch the Job

	return ctrl.Result{}, nil
}
