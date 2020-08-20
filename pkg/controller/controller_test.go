package controller

import (
	"context"
	"fmt"

	mock_cache "github.com/bakito/batch-job-controller/pkg/mocks/cache"
	mock_client "github.com/bakito/batch-job-controller/pkg/mocks/client"
	mock_logr "github.com/bakito/batch-job-controller/pkg/mocks/logr"
	gm "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
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
			Ω(p.Create(event.CreateEvent{Meta: m1})).Should(BeTrue())
			Ω(p.Update(event.UpdateEvent{MetaNew: m1})).Should(BeTrue())
			Ω(p.Delete(event.DeleteEvent{Meta: m1})).Should(BeTrue())
			Ω(p.Generic(event.GenericEvent{Meta: m1})).Should(BeTrue())
		})
		It("should not match", func() {
			Ω(p.Create(event.CreateEvent{Meta: m2})).Should(BeFalse())
			Ω(p.Update(event.UpdateEvent{MetaNew: m2})).Should(BeFalse())
			Ω(p.Delete(event.DeleteEvent{Meta: m2})).Should(BeFalse())
			Ω(p.Generic(event.GenericEvent{Meta: m2})).Should(BeFalse())
		})
	})

	Context("Reconcile", func() {
		var (
			r          *PodReconciler
			mockCtrl   *gm.Controller //gomock struct
			mockCache  *mock_cache.MockCache
			mockClient *mock_client.MockClient
			mockLog    *mock_logr.MockLogger
		)
		BeforeEach(func() {
			mockCtrl = gm.NewController(GinkgoT())
			mockCache = mock_cache.NewMockCache(mockCtrl)
			mockClient = mock_client.NewMockClient(mockCtrl)
			mockLog = mock_logr.NewMockLogger(mockCtrl)
			r = &PodReconciler{}
			r.Cache = mockCache
			r.Client = mockClient
			r.Log = mockLog
		})
		It("should not find an entry", func() {
			mockLog.EXPECT().WithValues(gm.Any()).Return(mockLog)
			mockClient.EXPECT().Get(gm.Any(), gm.Any(), gm.AssignableToTypeOf(&corev1.Pod{})).Return(k8serrors.NewNotFound(schema.GroupResource{Group: "", Resource: ""}, ""))

			result, err := r.Reconcile(ctrl.Request{})
			Ω(err).ShouldNot(HaveOccurred())
			Ω(result).ShouldNot(BeNil())
			Ω(result.Requeue).Should(BeFalse())
		})
		It("should return an error", func() {
			mockLog.EXPECT().WithValues(gm.Any()).Return(mockLog)
			mockLog.EXPECT().Error(gm.Any(), gm.Any())
			mockClient.EXPECT().Get(gm.Any(), gm.Any(), gm.AssignableToTypeOf(&corev1.Pod{})).Return(fmt.Errorf(""))

			result, err := r.Reconcile(ctrl.Request{})
			Ω(err).Should(HaveOccurred())
			Ω(result).ShouldNot(BeNil())
			Ω(result.Requeue).Should(BeFalse())
		})
		It("should update cache on pod succeeded", func() {
			mockLog.EXPECT().WithValues(gm.Any()).Return(mockLog)
			mockLog.EXPECT().Error(gm.Any(), gm.Any())
			mockClient.EXPECT().Get(gm.Any(), gm.Any(), gm.AssignableToTypeOf(&corev1.Pod{})).
				Do(func(ctx context.Context, key client.ObjectKey, pod *corev1.Pod) error {
					pod.Status = corev1.PodStatus{
						Phase: corev1.PodSucceeded,
					}
					return nil
				})
			mockCache.EXPECT().PodTerminated(gm.Any(), gm.Any(), corev1.PodSucceeded)

			result, err := r.Reconcile(ctrl.Request{})
			Ω(err).ShouldNot(HaveOccurred())
			Ω(result).ShouldNot(BeNil())
			Ω(result.Requeue).Should(BeFalse())
		})
		It("should update cache on pod failed", func() {
			mockLog.EXPECT().WithValues(gm.Any()).Return(mockLog)
			mockLog.EXPECT().Error(gm.Any(), gm.Any())
			mockClient.EXPECT().Get(gm.Any(), gm.Any(), gm.AssignableToTypeOf(&corev1.Pod{})).
				Do(func(ctx context.Context, key client.ObjectKey, pod *corev1.Pod) error {
					pod.Status = corev1.PodStatus{
						Phase: corev1.PodFailed,
					}
					return nil
				})
			mockCache.EXPECT().PodTerminated(gm.Any(), gm.Any(), corev1.PodFailed)

			result, err := r.Reconcile(ctrl.Request{})
			Ω(err).ShouldNot(HaveOccurred())
			Ω(result).ShouldNot(BeNil())
			Ω(result.Requeue).Should(BeFalse())
		})
		It("should return error on update cache error", func() {
			mockLog.EXPECT().WithValues(gm.Any()).Return(mockLog)
			mockLog.EXPECT().Error(gm.Any(), gm.Any())
			mockClient.EXPECT().Get(gm.Any(), gm.Any(), gm.AssignableToTypeOf(&corev1.Pod{})).
				Do(func(ctx context.Context, key client.ObjectKey, pod *corev1.Pod) error {
					pod.Status = corev1.PodStatus{
						Phase: corev1.PodSucceeded,
					}
					return nil
				})
			mockCache.EXPECT().PodTerminated(gm.Any(), gm.Any(), corev1.PodSucceeded).Return(fmt.Errorf("error"))

			result, err := r.Reconcile(ctrl.Request{})
			Ω(err).Should(HaveOccurred())
			Ω(result).ShouldNot(BeNil())
			Ω(result.Requeue).Should(BeFalse())
		})
	})
})
