package controller

import (
	"bytes"
	"context"
	"errors"
	"io"

	"github.com/bakito/batch-job-controller/pkg/lifecycle"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// LabelOwner owner label
	LabelOwner = "batch-job-controller.bakito.github.com/owner"
	// LabelExecutionID execution id label
	LabelExecutionID = "batch-job-controller.bakito.github.com/execution-id"
)

// PodReconciler reconciler
type PodReconciler struct {
	client.Client
	kubeClient *kubernetes.Clientset
	Controller lifecycle.Controller
}

// SetupWithManager setup
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	clientset, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		return err
	}
	r.kubeClient = clientset
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Watches(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForObject{}).
		WithEventFilter(&podPredicate{}).
		Complete(r)
}

// Reconcile reconcile pods
func (r *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	podLog := log.FromContext(ctx)
	pod := &corev1.Pod{}
	err := r.Get(ctx, req.NamespacedName, pod)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}

		podLog.Error(err, "unexpected error")
		return reconcile.Result{}, err
	}

	executionID := pod.GetLabels()[LabelExecutionID]
	node := pod.Spec.NodeName

	if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
		containerLogs := make(map[string]string)
		if r.Controller.Config().SavePodLog {
			for _, c := range pod.Spec.Containers {
				if l, err := r.getPodLog(ctx, pod.Namespace, pod.Name, c.Name); err != nil {
					podLog.WithValues("container", c.Name).Info("could not get log if container")
				} else {
					containerLogs[c.Name] = l
				}
			}

		}
		if err := r.Controller.PodTerminated(executionID, node, pod.Status.Phase, containerLogs); err != nil {
			if !errors.Is(err, &lifecycle.ExecutionIDNotFound{}) {
				podLog.Error(err, "unexpected error")
				return reconcile.Result{}, err
			}
		}
	}

	return reconcile.Result{}, nil
}

type podPredicate struct{}

func (podPredicate) Create(e event.CreateEvent) bool {
	return matches(e.Object)
}

func (podPredicate) Update(e event.UpdateEvent) bool {
	return matches(e.ObjectNew)
}

func (podPredicate) Delete(e event.DeleteEvent) bool {
	return matches(e.Object)
}

func (podPredicate) Generic(e event.GenericEvent) bool {
	return matches(e.Object)
}

func matches(m metav1.Object) bool {
	return m.GetLabels()[LabelExecutionID] != "" && m.GetLabels()[LabelOwner] != ""
}

func (r *PodReconciler) getPodLog(ctx context.Context, namespace string, name string, containerName string) (string, error) {
	podLogOpts := corev1.PodLogOptions{
		Container: containerName,
	}
	req := r.kubeClient.CoreV1().Pods(namespace).GetLogs(name, &podLogOpts)
	podLogs, err := req.Stream(ctx)
	if err != nil {
		return "", err
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", err
	}
	str := buf.String()

	return str, nil
}
