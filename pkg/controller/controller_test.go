package controller

import (
	"context"
	"fmt"

	mock_client "github.com/bakito/batch-job-controller/pkg/mocks/client"
	mock_lifecycle "github.com/bakito/batch-job-controller/pkg/mocks/lifecycle"
	mock_logr "github.com/bakito/batch-job-controller/pkg/mocks/logr"
	gm "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var _ = Describe("Controller", func() {
	Context("podPredicate", func() {
		var (
			m1 client.Object
			m2 client.Object
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
			Ω(p.Create(event.CreateEvent{Object: m1})).Should(BeTrue())
			Ω(p.Update(event.UpdateEvent{ObjectNew: m1})).Should(BeTrue())
			Ω(p.Delete(event.DeleteEvent{Object: m1})).Should(BeTrue())
			Ω(p.Generic(event.GenericEvent{Object: m1})).Should(BeTrue())
		})
		It("should not match", func() {
			Ω(p.Create(event.CreateEvent{Object: m2})).Should(BeFalse())
			Ω(p.Update(event.UpdateEvent{ObjectNew: m2})).Should(BeFalse())
			Ω(p.Delete(event.DeleteEvent{Object: m2})).Should(BeFalse())
			Ω(p.Generic(event.GenericEvent{Object: m2})).Should(BeFalse())
		})
	})

	Context("Reconcile", func() {
		var (
			r              *PodReconciler
			mockCtrl       *gm.Controller // gomock struct
			mockController *mock_lifecycle.MockController
			mockClient     *mock_client.MockClient
			mockLog        *mock_logr.MockLogger
			ctx            context.Context
		)
		BeforeEach(func() {
			mockCtrl = gm.NewController(GinkgoT())
			mockController = mock_lifecycle.NewMockController(mockCtrl)
			mockClient = mock_client.NewMockClient(mockCtrl)
			mockLog = mock_logr.NewMockLogger(mockCtrl)
			ctx = log.IntoContext(context.TODO(), mockLog)
			r = &PodReconciler{}
			r.Controller = mockController
			r.Client = mockClient
		})
		It("should not find an entry", func() {
			mockLog.EXPECT().WithValues(gm.Any()).Return(mockLog)
			mockClient.EXPECT().Get(gm.Any(), gm.Any(), gm.AssignableToTypeOf(&corev1.Pod{})).Return(k8serrors.NewNotFound(schema.GroupResource{Group: "", Resource: ""}, ""))

			result, err := r.Reconcile(ctx, ctrl.Request{})
			Ω(err).ShouldNot(HaveOccurred())
			Ω(result).ShouldNot(BeNil())
			Ω(result.Requeue).Should(BeFalse())
		})
		It("should return an error", func() {
			mockLog.EXPECT().WithValues(gm.Any()).Return(mockLog)
			mockLog.EXPECT().Error(gm.Any(), gm.Any())
			mockClient.EXPECT().Get(gm.Any(), gm.Any(), gm.AssignableToTypeOf(&corev1.Pod{})).Return(fmt.Errorf(""))

			result, err := r.Reconcile(ctx, ctrl.Request{})
			Ω(err).Should(HaveOccurred())
			Ω(result).ShouldNot(BeNil())
			Ω(result.Requeue).Should(BeFalse())
		})
		It("should update controller on pod succeeded", func() {
			mockLog.EXPECT().WithValues(gm.Any()).Return(mockLog)
			mockClient.EXPECT().Get(gm.Any(), gm.Any(), gm.AssignableToTypeOf(&corev1.Pod{})).
				Do(func(ctx context.Context, key client.ObjectKey, pod *corev1.Pod) error {
					pod.Status = corev1.PodStatus{
						Phase: corev1.PodSucceeded,
					}
					return nil
				})
			mockController.EXPECT().PodTerminated(gm.Any(), gm.Any(), corev1.PodSucceeded)

			result, err := r.Reconcile(ctx, ctrl.Request{})
			Ω(err).ShouldNot(HaveOccurred())
			Ω(result).ShouldNot(BeNil())
			Ω(result.Requeue).Should(BeFalse())
		})
		It("should update controller on pod failed", func() {
			mockLog.EXPECT().WithValues(gm.Any()).Return(mockLog)
			mockClient.EXPECT().Get(gm.Any(), gm.Any(), gm.AssignableToTypeOf(&corev1.Pod{})).
				Do(func(ctx context.Context, key client.ObjectKey, pod *corev1.Pod) error {
					pod.Status = corev1.PodStatus{
						Phase: corev1.PodFailed,
					}
					return nil
				})
			mockController.EXPECT().PodTerminated(gm.Any(), gm.Any(), corev1.PodFailed)

			result, err := r.Reconcile(ctx, ctrl.Request{})
			Ω(err).ShouldNot(HaveOccurred())
			Ω(result).ShouldNot(BeNil())
			Ω(result.Requeue).Should(BeFalse())
		})
		It("should return error on update controller error", func() {
			mockLog.EXPECT().WithValues(gm.Any()).Return(mockLog)
			mockLog.EXPECT().Error(gm.Any(), gm.Any())
			mockClient.EXPECT().Get(gm.Any(), gm.Any(), gm.AssignableToTypeOf(&corev1.Pod{})).
				Do(func(ctx context.Context, key client.ObjectKey, pod *corev1.Pod) error {
					pod.Status = corev1.PodStatus{
						Phase: corev1.PodSucceeded,
					}
					return nil
				})
			mockController.EXPECT().PodTerminated(gm.Any(), gm.Any(), corev1.PodSucceeded).Return(fmt.Errorf("error"))

			result, err := r.Reconcile(ctx, ctrl.Request{})
			Ω(err).Should(HaveOccurred())
			Ω(result).ShouldNot(BeNil())
			Ω(result.Requeue).Should(BeFalse())
		})
	})
})
