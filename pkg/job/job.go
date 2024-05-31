package job

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/bakito/batch-job-controller/pkg/config"
	"github.com/bakito/batch-job-controller/pkg/controller"
	"github.com/bakito/batch-job-controller/pkg/http"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	envNodeName    = "NODE_NAME"
	envExecutionID = "EXECUTION_ID"
	envNamespace   = "NAMESPACE"
	// EnvCallbackServiceName env var name of the callback service name
	EnvCallbackServiceName = "CALLBACK_SERVICE_NAME"
	// EnvCallbackServicePort env var name of the callback service port
	EnvCallbackServicePort = "CALLBACK_SERVICE_PORT"
	// EnvCallbackServiceResultURL env var name of the callback service result endpoint
	EnvCallbackServiceResultURL = "CALLBACK_SERVICE_RESULT_URL"
	// EnvCallbackServiceFileURL env var name of the callback service file endpoint
	EnvCallbackServiceFileURL = "CALLBACK_SERVICE_FILE_URL"
	// EnvCallbackServiceEventURL env var name of the callback service event endpoint
	EnvCallbackServiceEventURL = "CALLBACK_SERVICE_EVENT_URL"
)

var (
	reservedEnvVars = map[string]bool{
		envNodeName:            true,
		envExecutionID:         true,
		envNamespace:           true,
		EnvCallbackServiceName: true,
		EnvCallbackServicePort: true,
	}

	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(corev1.AddToScheme(scheme))
}

// MatchingLabels the filter match labels for a job
func MatchingLabels(name string) client.MatchingLabels {
	return client.MatchingLabels{controller.LabelOwner: name}
}

// New create a new job
func New(cfg *config.Config, nodeName, id, callbackAddress string, owner runtime.Object, extender ...CustomPodEnv) (*corev1.Pod, error) {
	podName := cfg.PodName(nodeName, id)

	data := map[string]string{
		"Namespace":   cfg.Namespace,
		"ExecutionID": id,
		"NodeName":    nodeName,
	}
	tmpl, err := template.New("job-pod").Parse(cfg.JobPodTemplate)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return nil, err
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: make(map[string]string),
			Labels:      make(map[string]string),
		},
	}
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(buf.Bytes()), 20)
	err = decoder.Decode(pod)
	if err != nil {
		return nil, err
	}

	pod.ObjectMeta.Name = podName
	pod.ObjectMeta.Namespace = cfg.Namespace

	// assure correct labels
	pod.Labels[controller.LabelExecutionID] = id
	pod.Labels[controller.LabelOwner] = cfg.Name

	// assure correct node name
	pod.Spec.NodeName = nodeName

	// assure correct service account
	pod.Spec.ServiceAccountName = cfg.JobServiceAccount

	//assure correct image pull secrets
	pod.Spec.ImagePullSecrets = cfg.JobImagePullSecrets

	// assure restart policy is set to never
	pod.Spec.RestartPolicy = corev1.RestartPolicyNever

	// assure correct env
	for i := range pod.Spec.Containers {
		newEnv := mergeEnv(cfg, nodeName, id, callbackAddress, pod.Spec.Containers[i], extender)
		pod.Spec.Containers[i].Env = newEnv
	}
	for i := range pod.Spec.InitContainers {
		newEnv := mergeEnv(cfg, nodeName, id, callbackAddress, pod.Spec.InitContainers[i], extender)
		pod.Spec.InitContainers[i].Env = newEnv
	}

	if owner != nil {
		if mo, ok := owner.(metav1.Object); ok {
			_ = controllerutil.SetOwnerReference(mo, pod, scheme)
		}
	}

	return pod, err
}

func mergeEnv(cfg *config.Config, nodeName string, id string, callbackAddress string, container corev1.Container, extender []CustomPodEnv) []corev1.EnvVar {
	var newEnv []corev1.EnvVar
	for _, e := range container.Env {
		// keep all non reserved env variables
		if _, ok := reservedEnvVars[e.Name]; !ok {
			newEnv = append(newEnv, e)
		}
	}

	for _, e := range extender {
		newEnv = append(newEnv, e.ExtendEnv(cfg, nodeName, id, callbackAddress, container)...)
	}

	newEnv = append(newEnv, corev1.EnvVar{Name: envExecutionID, Value: id})
	newEnv = append(newEnv, corev1.EnvVar{Name: envNamespace, Value: cfg.Namespace})
	newEnv = append(newEnv, corev1.EnvVar{Name: envNodeName, Value: nodeName})
	newEnv = append(newEnv, corev1.EnvVar{Name: EnvCallbackServiceName, Value: callbackAddress})
	newEnv = append(newEnv, corev1.EnvVar{Name: EnvCallbackServicePort, Value: fmt.Sprintf("%d", cfg.CallbackServicePort)})
	newEnv = append(newEnv, corev1.EnvVar{
		Name:  EnvCallbackServiceResultURL,
		Value: fmt.Sprintf("http://%s:%d/report/%s/%s%s", callbackAddress, cfg.CallbackServicePort, nodeName, id, http.CallbackBaseResultSubPath),
	})
	newEnv = append(newEnv, corev1.EnvVar{
		Name:  EnvCallbackServiceFileURL,
		Value: fmt.Sprintf("http://%s:%d/report/%s/%s%s", callbackAddress, cfg.CallbackServicePort, nodeName, id, http.CallbackBaseFileSubPath),
	})
	newEnv = append(newEnv, corev1.EnvVar{
		Name:  EnvCallbackServiceEventURL,
		Value: fmt.Sprintf("http://%s:%d/report/%s/%s%s", callbackAddress, cfg.CallbackServicePort, nodeName, id, http.CallbackBaseEventSubPath),
	})

	return newEnv
}
