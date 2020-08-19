package config

import (
	"context"
	"fmt"
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	EnvHostname      = "HOSTNAME"
	EnvConfigMapName = "CONFIG_MAP_NAME"
	PodTemplateName  = "pod-template.yaml"
	ConfigFileName   = "config.yaml"
)

var (
	log = ctrl.Log.WithName("config")
)

func Get(namespace string, cl client.Reader) (*Config, error) {

	cm, err := configMap(namespace, cl)
	if err != nil {
		return nil, err
	}
	if c, ok := cm.Data[ConfigFileName]; ok {
		cfg := &Config{}
		decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(c), 20)
		err = decoder.Decode(cfg)

		if err != nil {
			return nil, fmt.Errorf("could not read config file %q in configmap %q: %v", ConfigFileName, os.Getenv(EnvConfigMapName), err)
		}

		if t, ok := cm.Data[PodTemplateName]; !ok {
			return nil, fmt.Errorf("could not find pod template %q in configmap %q", PodTemplateName, os.Getenv(EnvConfigMapName))
		} else {
			cfg.JobPodTemplate = t
		}

		cfg.Namespace = namespace

		cfg.Owner = findPodOwner(namespace, cl)

		return cfg, nil
	}
	return nil, fmt.Errorf("could not find config file %q in configmap %q", ConfigFileName, os.Getenv(EnvConfigMapName))
}

func configMap(namespace string, cl client.Reader) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{}
	err := cl.Get(context.TODO(), client.ObjectKey{Namespace: namespace, Name: os.Getenv(EnvConfigMapName)}, cm)
	if err != nil {
		err = fmt.Errorf("error getting configmap %q: %v", EnvConfigMapName, err)
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

func findOwner(obj runtime.Object, namespace string, name string, cl client.Reader) (string, runtime.Object) {
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
				return findOwner(us, namespace, or.Name, cl)
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
