package cron

import (
	"context"
	"github.com/bakito/batch-job-controller/pkg/config"
	"github.com/bakito/batch-job-controller/pkg/job"
	mock_cache "github.com/bakito/batch-job-controller/pkg/mocks/cache"
	mock_client "github.com/bakito/batch-job-controller/pkg/mocks/client"
	gm "github.com/golang/mock/gomock"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Cron", func() {
	var (
		cj         *cronJob
		mockCtrl   *gm.Controller //gomock struct
		mockClient *mock_client.MockClient
		mockCache  *mock_cache.MockCache
		namespace  string
		configName string
	)
	BeforeEach(func() {
		mockCtrl = gm.NewController(GinkgoT())
		mockClient = mock_client.NewMockClient(mockCtrl)
		mockCache = mock_cache.NewMockCache(mockCtrl)
		namespace = uuid.New().String()
		configName = uuid.New().String()
		cfg := &config.Config{
			Name:      configName,
			Namespace: namespace,
		}
		cj = Job().(*cronJob)
		cj.InjectCache(mockCache)
		cj.InjectConfig(cfg)
		cj.InjectClient(mockClient)

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
		var (
			nodeSelector map[string]string
		)
		BeforeEach(func() {
			nodeSelector = map[string]string{"foo": "bar"}
			cj.cfg.JobNodeSelector = nodeSelector
			cj.cfg.JobPodTemplate = "kind: Pod"
			mockCache.EXPECT().NewExecution()
			mockCache.EXPECT().AllAdded(gm.Any())
			mockCache.EXPECT().AddPod(gm.Any())
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
			cj.startPods()
		})
	})

	Context("startPods - already running", func() {
		It("should not start all pods", func() {
			cj.running = true
			cj.startPods()
		})
	})

	Context("CreatePod", func() {

		var (
			pj       *podJob
			id       string
			nodeName string
		)
		BeforeEach(func() {

			id = uuid.New().String()
			nodeName = uuid.New().String()
			pj = &podJob{
				pod:      &corev1.Pod{},
				client:   mockClient,
				id:       id,
				nodeName: nodeName,
			}
		})
		It("should create a pod", func() {
			mockClient.EXPECT().Create(gm.Any(), pj.pod)
			pj.CreatePod()
		})
		It("should return the id", func() {
			Ω(pj.ID()).Should(Equal(id))
		})
		It("should return the nodeName", func() {
			Ω(pj.Node()).Should(Equal(nodeName))
		})
	})
})
