package controller

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/bakito/batch-job-controller/pkg/lifecycle"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

var clog = ctrl.Log.WithName("pod-controller")

// PodReconciler reconciler
type PodReconciler struct {
	client.Client
	coreClient corev1client.CoreV1Interface
	Controller lifecycle.Controller
}

// SetupWithManager setup
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	clientset, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		return err
	}
	r.coreClient = clientset.CoreV1()
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
		if r.Controller.Config().SavePodLog && pod.DeletionTimestamp == nil {
			r.savePodLogs(ctx, pod, executionID)
		}
		if err := r.Controller.PodTerminated(executionID, node, pod.Status.Phase); err != nil {
			if !errors.Is(err, &lifecycle.ExecutionIDNotFound{}) {
				podLog.Error(err, "unexpected error")
				return reconcile.Result{}, err
			}
		}
	}

	return reconcile.Result{}, nil
}

func (r *PodReconciler) savePodLogs(ctx context.Context, pod *corev1.Pod, executionID string) {
	for _, c := range pod.Spec.Containers {
		clog := clog.WithValues("node", pod.Spec.NodeName, "id", executionID, "container", c.Name)
		if l, err := r.getPodLog(ctx, pod.Namespace, pod.Name, c.Name); err != nil {
			clog.Error(err, "could not get log of container")
		} else {
			if err := r.savePodLog(pod.Spec.NodeName, executionID, c.Name, l); err != nil {
				clog.Error(err, "error saving container log file")
			} else {
				clog.Info("saved container log file")
			}
		}
	}
}

func (r *PodReconciler) getPodLog(ctx context.Context, namespace string, name string, containerName string) (string, error) {
	podLogOpts := corev1.PodLogOptions{
		Container: containerName,
	}
	req := r.coreClient.Pods(namespace).GetLogs(name, &podLogOpts)
	podLogs, err := req.Stream(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = podLogs.Close() }()

	buf := new(bytes.Buffer)
	if _, err = io.Copy(buf, podLogs); err != nil {
		return "", err
	}
	str := buf.String()

	return str, nil
}

func (r *PodReconciler) savePodLog(node string, executionID string, name string, data string) error {
	if err := r.Controller.Config().MkReportDir(executionID); err != nil {
		return err
	}
	fileName := r.Controller.Config().ReportFileName(executionID, fmt.Sprintf("%s-container-%s.log", node, name))
	return ioutil.WriteFile(fileName, []byte(data), 0o600)
}
