package controller

import (
	"sigs.k8s.io/controller-runtime/pkg/event"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Controller", func() {
	Context("podPredicate", func() {
		var (
			m1 metav1.Object
			m2 metav1.Object
			p  *podPredicate
		)
		BeforeEach(func() {
			m1 = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						LabelExecutionID: "foo",
						LabelOwner:       "bar",
					},
				},
			}
			m2 = &corev1.Pod{}
			p = &podPredicate{}
		})
		It("should match", func() {
			Ω(p.Create(event.CreateEvent{Meta: m1})).To(BeTrue())
			Ω(p.Update(event.UpdateEvent{MetaNew: m1})).To(BeTrue())
			Ω(p.Delete(event.DeleteEvent{Meta: m1})).To(BeTrue())
			Ω(p.Generic(event.GenericEvent{Meta: m1})).To(BeTrue())
		})
		It("should not match", func() {
			Ω(p.Create(event.CreateEvent{Meta: m2})).To(BeFalse())
			Ω(p.Update(event.UpdateEvent{MetaNew: m2})).To(BeFalse())
			Ω(p.Delete(event.DeleteEvent{Meta: m2})).To(BeFalse())
			Ω(p.Generic(event.GenericEvent{Meta: m2})).To(BeFalse())
		})
	})
})
