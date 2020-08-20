package controller

import (
	"context"
	"github.com/bakito/batch-job-controller/pkg/lifecycle"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
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
	Log   logr.Logger
	Cache lifecycle.Cache
}

// SetupWithManager setup
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Watches(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForObject{}).
		WithEventFilter(&podPredicate{}).
		Complete(r)
}

// Reconcile reconcile pods
func (r *PodReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	podLog := r.Log.WithValues("pod", req.NamespacedName)
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

	switch pod.Status.Phase {
	case corev1.PodSucceeded:
		err = r.Cache.PodTerminated(executionID, node, pod.Status.Phase)
	case corev1.PodFailed:
		err = r.Cache.PodTerminated(executionID, node, pod.Status.Phase)
	}
	if err != nil {
		_, ok := err.(lifecycle.ExecutionIDNotFound)
		if !ok {
			podLog.Error(err, "error updating cache")
		}
	}

	return reconcile.Result{}, nil
}

type podPredicate struct {
}

func (podPredicate) Create(e event.CreateEvent) bool {
	return matches(e.Meta)
}

func (podPredicate) Update(e event.UpdateEvent) bool {
	return matches(e.MetaNew)
}

func (podPredicate) Delete(e event.DeleteEvent) bool {
	return matches(e.Meta)
}

func (podPredicate) Generic(e event.GenericEvent) bool {
	return matches(e.Meta)
}

func matches(m metav1.Object) bool {
	return m.GetLabels()[LabelExecutionID] != "" && m.GetLabels()[LabelOwner] != ""
}
