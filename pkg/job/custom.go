package job

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/bakito/batch-job-controller/pkg/config"
)

// CustomPodEnv interface.
type CustomPodEnv interface {
	// ExtendEnv extend the env for the job pod
	ExtendEnv(cfg *config.Config, nodeName string, id string, serviceIP string, containers corev1.Container) []corev1.EnvVar
}
