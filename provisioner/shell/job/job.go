package job

import (
	"fmt"
	"time"

	"github.com/forge-build/forge/pkg/kube"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	buildv1 "github.com/forge-build/forge/api/v1alpha1"
	"github.com/forge-build/forge/provisioner/shell"
)

const (
	containerName = "shell-provisioner"
)

type ShellJobBuilder struct {
	uuid                     string
	name                     string
	namespace                string
	buildNamespace           string
	scriptToRun              string
	scriptToRunRef           string
	sshCredentialsSecretName string

	repo string
	tag  string

	ttl                      *time.Duration
	timeout                  time.Duration
	backoffLimit             int32
	tolerations              []corev1.Toleration
	nodeSelector             map[string]string
	annotations              map[string]string
	podTemplateLabels        map[string]string
	podSecurityContext       *corev1.PodSecurityContext
	containerSecurityContext *corev1.SecurityContext
	podPriorityClassName     string
	resourceRequirements     corev1.ResourceRequirements
}

func (s *ShellJobBuilder) WithUUID(n string) *ShellJobBuilder {
	s.uuid = n
	return s
}

func (s *ShellJobBuilder) WithBuildName(n string) *ShellJobBuilder {
	s.name = n
	return s
}

func (s *ShellJobBuilder) WithBuildNamespace(n string) *ShellJobBuilder {
	s.buildNamespace = n
	return s
}

func (s *ShellJobBuilder) WithScriptToRun(r string) *ShellJobBuilder {
	s.scriptToRun = r
	return s
}

func (s *ShellJobBuilder) WithScriptToRunRef(r string) *ShellJobBuilder {
	s.scriptToRunRef = r
	return s
}

func (s *ShellJobBuilder) WithSSHCredentialsSecretName(name string) *ShellJobBuilder {
	s.sshCredentialsSecretName = name
	return s
}

func (s *ShellJobBuilder) WithRepo(r string) *ShellJobBuilder {
	s.repo = r
	return s
}

func (s *ShellJobBuilder) WithTag(t string) *ShellJobBuilder {
	s.tag = t
	return s
}

func (s *ShellJobBuilder) WithTimeout(timeout time.Duration) *ShellJobBuilder {
	s.timeout = timeout
	return s
}

func (s *ShellJobBuilder) WithBackOffLimit(backOffLimit int32) *ShellJobBuilder {
	s.backoffLimit = backOffLimit
	return s
}

func (s *ShellJobBuilder) WithTTL(ttl *time.Duration) *ShellJobBuilder {
	s.ttl = ttl
	return s
}

func (s *ShellJobBuilder) WithTolerations(tolerations []corev1.Toleration) *ShellJobBuilder {
	s.tolerations = tolerations
	return s
}

func (s *ShellJobBuilder) WithAnnotations(annotations map[string]string) *ShellJobBuilder {
	s.annotations = annotations
	return s
}

func (s *ShellJobBuilder) WithNodeSelector(nodeSelector map[string]string) *ShellJobBuilder {
	s.nodeSelector = nodeSelector
	return s
}

func (s *ShellJobBuilder) WithPodSecurityContext(podSecurityContext *corev1.PodSecurityContext) *ShellJobBuilder {
	s.podSecurityContext = podSecurityContext
	return s
}

func (s *ShellJobBuilder) WithPodPriorityClassName(podPriorityClassName string) *ShellJobBuilder {
	s.podPriorityClassName = podPriorityClassName
	return s
}

func (s *ShellJobBuilder) WithSecurityContext(securityContext *corev1.SecurityContext) *ShellJobBuilder {
	s.containerSecurityContext = securityContext
	return s
}

func (s *ShellJobBuilder) WithPodTemplateLabels(podTemplateLabels map[string]string) *ShellJobBuilder {
	s.podTemplateLabels = podTemplateLabels
	return s
}

func (s *ShellJobBuilder) WithNamespace(ns string) *ShellJobBuilder {
	s.namespace = ns
	return s
}

func (s *ShellJobBuilder) WithResourceRequirements(r corev1.ResourceRequirements) *ShellJobBuilder {
	s.resourceRequirements = r
	return s
}

func NewShellJobBuilder() *ShellJobBuilder {
	return &ShellJobBuilder{}
}

func (s *ShellJobBuilder) Build() (*batchv1.Job, error) {
	templateSpec := s.getPodSpec()

	jobLabels := map[string]string{
		buildv1.ManagedByLabel:      shell.ForgeProvisionerShellName,
		buildv1.BuildNameLabel:      s.name,
		buildv1.ProvisionerIDLabel:  s.uuid,
		buildv1.BuildNamespaceLabel: s.buildNamespace,
	}
	podTemplateLabels := make(map[string]string)
	for k, v := range jobLabels {
		podTemplateLabels[k] = v
	}

	jobSpec := batchv1.JobSpec{
		BackoffLimit:          ptr.To(s.backoffLimit), // number of retries before marking job as failed.
		Completions:           ptr.To(int32(1)),
		ActiveDeadlineSeconds: DurationSecondsPtr(s.timeout),
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels:      podTemplateLabels,
				Annotations: s.annotations,
			},
			Spec: templateSpec,
		},
	}

	if s.ttl != nil {
		if s.ttl.Seconds() > 0 {
			jobSpec.TTLSecondsAfterFinished = ptr.To(int32(s.ttl.Seconds()))
		}
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   s.namespace,
			Labels:      jobLabels,
			Annotations: map[string]string{},
		},
		Spec: jobSpec,
	}
	job.SetName(GetShellJobName(s.name))

	return job, nil
}

func (s *ShellJobBuilder) getPodSpec() corev1.PodSpec {
	shelljobImageRef := s.GetImageRef()

	var containers []corev1.Container
	var env []corev1.EnvVar

	env = append(env, corev1.EnvVar{
		Name: "POD_NAMESPACE",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.namespace",
			},
		},
	})

	volumes := make([]corev1.Volume, 0)
	volumeMounts := make([]corev1.VolumeMount, 0)
	// TODO add volumes
	//for _, secret := range s.pullSecrets {
	//	name := fmt.Sprintf("%s-%s", pullSecretNamePrefix, secret)
	//	mountPath := fmt.Sprintf("%s/%s", MountPathPrefix, secret)
	//
	//	volumes = append(volumes, corev1.Volume{
	//		Name: name,
	//		VolumeSource: corev1.VolumeSource{
	//			Secret: &corev1.SecretVolumeSource{
	//				SecretName: secret,
	//			},
	//		},
	//	})
	//
	//	volumeMounts = append(volumeMounts, corev1.VolumeMount{
	//		Name:      name,
	//		ReadOnly:  true,
	//		MountPath: mountPath,
	//	})
	//}

	args := s.getArgs()

	containers = append(
		containers,
		corev1.Container{
			Name:                     containerName,
			Image:                    shelljobImageRef,
			ImagePullPolicy:          corev1.PullIfNotPresent,
			TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
			Env:                      env,
			Args:                     args,
			VolumeMounts:             volumeMounts,
			Resources:                s.resourceRequirements,
		},
	)

	return corev1.PodSpec{
		ServiceAccountName: shell.ForgeProvisionerShellName,
		Volumes:            volumes,
		Affinity:           LinuxNodeAffinity(),
		RestartPolicy:      corev1.RestartPolicyNever,
		Containers:         containers,
		SecurityContext:    &corev1.PodSecurityContext{},
	}
}

func DurationSecondsPtr(d time.Duration) *int64 {
	if d > 0 {
		return ptr.To(int64(d.Seconds()))
	}
	return nil
}

func (s *ShellJobBuilder) getArgs() []string {
	if s.scriptToRunRef != "" {
		args := []string{
			"--namespace",
			s.buildNamespace,
			"--run-script-ref",
			s.scriptToRunRef,
			"--ssh-credentials-secret-name",
			s.sshCredentialsSecretName,
		}

		return args
	}
	return []string{
		"--namespace",
		s.buildNamespace,
		"--run-script",
		s.scriptToRun,
		"--ssh-credentials-secret-name",
		s.sshCredentialsSecretName,
	}
}

func GetShellJobName(buildName string) string {
	return fmt.Sprintf("forge-provisioner-shell-%s", kube.ComputeHash(buildName))
}

//
//func constructEnvVarSourceFromSecret(envName, secretName, secretKey string) (res corev1.EnvVar) {
//	res = corev1.EnvVar{
//		Name: envName,
//		ValueFrom: &corev1.EnvVarSource{
//			SecretKeyRef: &corev1.SecretKeySelector{
//				LocalObjectReference: corev1.LocalObjectReference{
//					Name: secretName,
//				},
//				Key:      secretKey,
//				Optional: ptr.To(true),
//			},
//		},
//	}
//	return
//}

// GetImageRef returns upstream Trivy container image reference.
func (s *ShellJobBuilder) GetImageRef() string {
	return fmt.Sprintf("%s:%s", s.repo, s.tag)
}

// LinuxNodeAffinity constructs a new Affinity resource with linux supported nodes.
func LinuxNodeAffinity() *corev1.Affinity {
	return &corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{
					{
						MatchExpressions: []corev1.NodeSelectorRequirement{
							{
								Key:      "kubernetes.io/os",
								Operator: corev1.NodeSelectorOpIn,
								Values:   []string{"linux"},
							},
						},
					},
				},
			},
		},
	}
}
