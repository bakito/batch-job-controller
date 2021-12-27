package config

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// EnvHostname hostname env variable
	EnvHostname = "HOSTNAME"
	// EnvConfigMapName configmap name env variable
	EnvConfigMapName = "CONFIG_MAP_NAME"
	// EnvPodIP the controller pod's IP
	EnvPodIP = "POD_IP"
	// EnvDevMode enable dev mode
	EnvDevMode = "DEV_MODE"
	// EnvReportDirectory override for report directory
	EnvReportDirectory = "REPORT_DIRECTORY"

	// PodTemplateName key of the pod template in the configmap
	PodTemplateName = "pod-template.yaml"
	// ConfigFileName key of the config yaml file in the configmap
	ConfigFileName = "config.yaml"

	// LabelVersion version label
	LabelVersion = "version"
	// LabelName name label
	LabelName = "name"
	// LabelPoolSize poolSize label
	LabelPoolSize = "poolSize"
	// LabelReportHistory reportHistory label
	LabelReportHistory = "reportHistory"
)

var log = ctrl.Log.WithName("config")

// Get read the config from the configmap
func Get(namespace string, config *rest.Config, scheme *runtime.Scheme) (*Config, error) {
	apiReader, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}
	return getInternal(namespace, apiReader)
}

func getInternal(namespace string, apiReader client.Reader) (*Config, error) {
	// Get read the config from the configmap
	cm, err := configMap(namespace, apiReader)
	if err != nil {
		return nil, err
	}
	if c, ok := cm.Data[ConfigFileName]; ok {
		cfg := &Config{}
		decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(c), 20)
		err = decoder.Decode(cfg)

		if err != nil {
			return nil, fmt.Errorf("could not read config file %q in configmap %q: %w", ConfigFileName, os.Getenv(EnvConfigMapName), err)
		}

		if t, ok := cm.Data[PodTemplateName]; ok {
			cfg.JobPodTemplate = t
		} else {
			return nil, fmt.Errorf("could not find pod template %q in configmap %q", PodTemplateName, os.Getenv(EnvConfigMapName))
		}

		cfg.Namespace = namespace

		cfg.Owner = findPodOwner(namespace, apiReader)

		if cfg.StartupDelay == 0 {
			cfg.StartupDelay = 10 * time.Second
		}

		cfg.DevMode = IsDevMode()
		if cfg.DevMode {
			log.Info("DEV MODE ENABLED!!!")
		}

		if dir, ok := os.LookupEnv(EnvReportDirectory); ok {
			cfg.ReportDirectory = dir
			log.WithValues("env", EnvReportDirectory, "reportDirectory", dir).Info("override report directory from env")
		}

		return cfg, nil
	}
	return nil, fmt.Errorf("could not find config file %q in configmap %q", ConfigFileName, os.Getenv(EnvConfigMapName))
}

func IsDevMode() bool {
	return strings.EqualFold(os.Getenv(EnvDevMode), "true")
}

func configMap(namespace string, cl client.Reader) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{}
	err := cl.Get(context.TODO(), client.ObjectKey{Namespace: namespace, Name: os.Getenv(EnvConfigMapName)}, cm)
	if err != nil {
		err = fmt.Errorf("error getting configmap %q: %w", EnvConfigMapName, err)
	}
	return cm, err
}

func findPodOwner(namespace string, cl client.Reader) runtime.Object {
	pod := &corev1.Pod{}
	log.WithValues("kind", "Pod", "name", os.Getenv(EnvHostname)).Info("looking for owner of current pod")
	name, owner := findOwner(pod, namespace, os.Getenv(EnvHostname), cl)
	if owner != nil {
		log.WithValues(
			"kind", owner.GetObjectKind().GroupVersionKind().Kind,
			"name", name,
		).Info("found owner for pods")
	}
	return owner
}

func findOwner(obj client.Object, namespace string, name string, cl client.Reader) (string, runtime.Object) {
	err := cl.Get(context.TODO(), client.ObjectKey{Namespace: namespace, Name: name}, obj)
	if err == nil {
		if ob, ok := obj.(metav1.Object); ok {
			for _, or := range ob.GetOwnerReferences() {
				var us runtime.Unstructured = &unstructured.Unstructured{
					Object: map[string]interface{}{
						"kind":       or.Kind,
						"apiVersion": or.APIVersion,
					},
				}
				return findOwner(us.(client.Object), namespace, or.Name, cl)
			}
			return name, obj
		}
	} else {
		log.WithValues(
			"kind", obj.GetObjectKind().GroupVersionKind().Kind,
			"name", name,
		).V(4).Info("error finding owner")
	}
	return name, nil
}
