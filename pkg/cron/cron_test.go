package cron

import (
	"context"
	"fmt"
	"os"

	"github.com/bakito/batch-job-controller/pkg/config"
	"github.com/bakito/batch-job-controller/pkg/job"
	mock_client "github.com/bakito/batch-job-controller/pkg/mocks/client"
	mock_lifecycle "github.com/bakito/batch-job-controller/pkg/mocks/lifecycle"
	mock_logr "github.com/bakito/batch-job-controller/pkg/mocks/logr"
	"github.com/go-logr/logr"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gm "go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Cron", func() {
	var (
		cj             *cronJob
		mockCtrl       *gm.Controller // gomock struct
		mockClient     *mock_client.MockClient
		mockController *mock_lifecycle.MockController
		mockSink       *mock_logr.MockLogSink
		namespace      string
		configName     string
		id             string
	)
	BeforeEach(func() {
		mockCtrl = gm.NewController(GinkgoT())
		mockClient = mock_client.NewMockClient(mockCtrl)
		mockController = mock_lifecycle.NewMockController(mockCtrl)
		mockSink = mock_logr.NewMockLogSink(mockCtrl)
		namespace = uuid.New().String()
		configName = uuid.New().String()
		id = uuid.New().String()
		cfg := &config.Config{
			Name:      configName,
			Namespace: namespace,
		}
		mockSink.EXPECT().Init(gm.Any())
		mockSink.EXPECT().Enabled(gm.Any()).AnyTimes().Return(true)
		log = logr.New(mockSink)
		cj = Job().(*cronJob)
		cj.InjectController(mockController)
		cj.InjectConfig(cfg)
		Ω(cj.InjectClient(mockClient)).ShouldNot(HaveOccurred())
	})
	Context("NeedLeaderElection", func() {
		It("should be true", func() {
			needLE := cj.NeedLeaderElection()
			Ω(needLE).Should(BeTrue())
		})
	})
	Context("deleteAll", func() {
		It("should delete all", func() {
			mockClient.EXPECT().DeleteAllOf(gm.Any(), gm.AssignableToTypeOf(&corev1.Pod{}), client.InNamespace(namespace), job.MatchingLabels(configName), client.PropagationPolicy(metav1.DeletePropagationBackground))

			err := cj.deleteAll(&corev1.Pod{})
			Ω(err).ShouldNot(HaveOccurred())
		})
	})

	Context("startPods", func() {
		var nodeSelector map[string]string
		BeforeEach(func() {
			nodeSelector = map[string]string{"foo": "bar"}
			cj.cfg.JobNodeSelector = nodeSelector
			cj.cfg.JobPodTemplate = "kind: Pod"
			mockController.EXPECT().NewExecution(1).Return(id)
			mockController.EXPECT().AllAdded(gm.Any())
			mockController.EXPECT().AddPod(gm.Any())
			mockClient.EXPECT().DeleteAllOf(gm.Any(), gm.Any(), gm.Any(), gm.Any(), gm.Any())
			mockClient.EXPECT().List(gm.Any(), gm.AssignableToTypeOf(&corev1.NodeList{}), client.MatchingLabels(nodeSelector)).
				Do(func(ctx context.Context, list *corev1.NodeList, opts ...client.ListOption) error {
					list.Items = []corev1.Node{
						{
							Spec: corev1.NodeSpec{
								Unschedulable: false,
							},
							Status: corev1.NodeStatus{
								Conditions: []corev1.NodeCondition{
									{
										Type:   corev1.NodeReady,
										Status: corev1.ConditionTrue,
									},
								},
							},
						},
					}
					return nil
				})
		})
		It("should start all pods with service IP for callback", func() {
			cj.cfg.CallbackServiceName = "any-service-name"
			mockClient.EXPECT().Get(gm.Any(), gm.Any(), gm.AssignableToTypeOf(&corev1.Service{}))
			mockSink.EXPECT().WithValues("id", id).Return(mockSink)
			mockSink.EXPECT().Info(gm.Any(), "deleting old job pods")
			mockSink.EXPECT().Info(gm.Any(), "executing job")
			cj.startPods()
		})
		It("should start all pods with pod IP for callback", func() {
			_ = os.Setenv(config.EnvPodIP, "1.2.3.4")
			defer func() {
				_ = os.Unsetenv(config.EnvPodIP)
			}()
			mockSink.EXPECT().WithValues("id", id).Return(mockSink)
			mockSink.EXPECT().Info(gm.Any(), "deleting old job pods")
			mockSink.EXPECT().Info(gm.Any(), "executing job")
			cj.startPods()
		})
	})

	Context("startPods - already running", func() {
		It("should not start all pods", func() {
			mockSink.EXPECT().Info(gm.Any(), "last cronjob still running")
			cj.running = true
			cj.startPods()
		})
	})

	Context("podJob", func() {
		var (
			pj       *podJob
			nodeName string
		)
		BeforeEach(func() {
			nodeName = uuid.New().String()
			pj = &podJob{
				pod:      &corev1.Pod{},
				client:   mockClient,
				id:       id,
				nodeName: nodeName,
			}
		})
		Context("CreatePod", func() {
			BeforeEach(func() {
				mockSink.EXPECT().Info(gm.Any(), "create pod", "node", nodeName)
			})
			It("should create a pod", func() {
				mockClient.EXPECT().Create(gm.Any(), pj.pod)
				pj.CreatePod()
			})
			It("should log an error a pod", func() {
				err := fmt.Errorf("some error")
				mockClient.EXPECT().Create(gm.Any(), pj.pod).Return(err)
				mockSink.EXPECT().Error(err, "unable to create pod", "node", nodeName)

				pj.CreatePod()
			})
		})
		It("should return the id", func() {
			Ω(pj.ID()).Should(Equal(id))
		})
		It("should return the nodeName", func() {
			Ω(pj.Node()).Should(Equal(nodeName))
		})
	})

	Context("isUsable", func() {
		var (
			node                  corev1.Node
			runOnUnscheduledNodes bool
		)
		BeforeEach(func() {
			node = corev1.Node{
				Spec:   corev1.NodeSpec{},
				Status: corev1.NodeStatus{},
			}
		})
		It("should return false if unschedulable", func() {
			node.Spec.Unschedulable = true
			runOnUnscheduledNodes = false
			Ω(isUsable(node, runOnUnscheduledNodes)).Should(BeFalse())
		})
		It("should return true if node ready", func() {
			node.Spec.Unschedulable = false
			node.Status.Conditions = []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionTrue}}
			runOnUnscheduledNodes = false
			Ω(isUsable(node, runOnUnscheduledNodes)).Should(BeTrue())
		})
		It("should return false if node not ready", func() {
			node.Spec.Unschedulable = false
			node.Status.Conditions = []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionFalse}}
			runOnUnscheduledNodes = false
			Ω(isUsable(node, runOnUnscheduledNodes)).Should(BeFalse())
		})
		It("should return false if no conditions are set", func() {
			node.Spec.Unschedulable = false
			runOnUnscheduledNodes = false
			Ω(isUsable(node, runOnUnscheduledNodes)).Should(BeFalse())
		})
	})
})
