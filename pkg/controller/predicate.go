package controller

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

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
