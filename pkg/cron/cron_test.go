package cron

import (
	"context"
	"fmt"

	"github.com/bakito/batch-job-controller/pkg/config"
	"github.com/bakito/batch-job-controller/pkg/job"
	mock_client "github.com/bakito/batch-job-controller/pkg/mocks/client"
	mock_lifecycle "github.com/bakito/batch-job-controller/pkg/mocks/lifecycle"
	mock_logr "github.com/bakito/batch-job-controller/pkg/mocks/logr"
	gm "github.com/golang/mock/gomock"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
		mockLog        *mock_logr.MockLogger
		namespace      string
		configName     string
		id             string
	)
	BeforeEach(func() {
		mockCtrl = gm.NewController(GinkgoT())
		mockClient = mock_client.NewMockClient(mockCtrl)
		mockController = mock_lifecycle.NewMockController(mockCtrl)
		mockLog = mock_logr.NewMockLogger(mockCtrl)
		namespace = uuid.New().String()
		configName = uuid.New().String()
		id = uuid.New().String()
		cfg := &config.Config{
			Name:      configName,
			Namespace: namespace,
		}
		log = mockLog
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
			mockClient.EXPECT().Get(gm.Any(), gm.Any(), gm.AssignableToTypeOf(&corev1.Service{}))
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
		It("should start all pods", func() {
			mockLog.EXPECT().WithValues("id", id).Return(mockLog)
			mockLog.EXPECT().Info("executing job")
			cj.startPods()
		})
	})

	Context("startPods - already running", func() {
		It("should not start all pods", func() {
			mockLog.EXPECT().Info("last cronjob still running")
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
				mockLog.EXPECT().Info("create pod", "node", nodeName)
			})
			It("should create a pod", func() {
				mockClient.EXPECT().Create(gm.Any(), pj.pod)
				pj.CreatePod()
			})
			It("should log an error a pod", func() {
				err := fmt.Errorf("some error")
				mockClient.EXPECT().Create(gm.Any(), pj.pod).Return(err)
				mockLog.EXPECT().Error(err, "unable to create pod", "node", nodeName)

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
