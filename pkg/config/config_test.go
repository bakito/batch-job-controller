package config

import (
	"context"
	"fmt"
	"os"

	mock_client "github.com/bakito/batch-job-controller/pkg/mocks/client"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gm "go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Config", func() {
	Context("Metrics", func() {
		var m *Metrics
		BeforeEach(func() {
			m = &Metrics{
				Prefix: "my_metric",
			}
		})
		It("should return a correct name", func() {
			Ω(m.NameFor("name")).Should(Equal("my_metric_name"))
		})
	})
	Context("PodName", func() {
		var (
			c        *Config
			name     string
			nodeName string
			node     string
			id       string
		)
		BeforeEach(func() {
			name = uuid.New().String()
			nodeName = uuid.New().String()
			node = nodeName + "." + uuid.New().String()
			id = uuid.New().String()
			c = &Config{
				Name: name,
			}
		})
		It("should return a correct name", func() {
			Ω(c.PodName(node, id)).Should(Equal(fmt.Sprintf("%s-job-%s-%s", name, nodeName, id)))
		})
	})

	Context("Get", func() {
		var (
			ctx        context.Context
			mockCtrl   *gm.Controller // gomock struct
			mockReader *mock_client.MockReader
			namespace  string
			cmName     string
			podName    string
			cmKey      client.ObjectKey
		)
		BeforeEach(func() {
			ctx = context.TODO()
			namespace = uuid.New().String()
			cmName = uuid.New().String()
			podName = uuid.New().String()
			mockCtrl = gm.NewController(GinkgoT())
			mockReader = mock_client.NewMockReader(mockCtrl)
			_ = os.Setenv(EnvConfigMapName, cmName)
			_ = os.Setenv(EnvHostname, podName)
			cmKey = client.ObjectKey{Namespace: namespace, Name: cmName}
		})

		Context("error", func() {
			It("should return an error returned by the reader", func() {
				mockReader.EXPECT().Get(ctx, cmKey, gm.AssignableToTypeOf(&corev1.ConfigMap{})).
					Return(fmt.Errorf("error"))

				c, err := getInternal(namespace, mockReader)
				Ω(c).Should(BeNil())
				Ω(err).Should(HaveOccurred())
				Ω(err.Error()).Should(ContainSubstring("error getting configmap"))
			})

			It("should return an error if no config is found", func() {
				mockReader.EXPECT().Get(ctx, cmKey, gm.AssignableToTypeOf(&corev1.ConfigMap{})).
					Do(func(ctx context.Context, key client.ObjectKey, cm *corev1.ConfigMap, opts ...client.GetOption) error {
						cm.Data = map[string]string{}
						return nil
					})

				c, err := getInternal(namespace, mockReader)
				Ω(c).Should(BeNil())
				Ω(err).Should(HaveOccurred())
				Ω(err.Error()).Should(ContainSubstring("could not find config file"))
			})

			It("should return an error if no config can not be parsed", func() {
				mockReader.EXPECT().Get(ctx, cmKey, gm.AssignableToTypeOf(&corev1.ConfigMap{})).
					Do(func(ctx context.Context, key client.ObjectKey, cm *corev1.ConfigMap, opts ...client.GetOption) error {
						cm.Data = map[string]string{
							ConfigFileName: "foo",
						}
						return nil
					})

				c, err := getInternal(namespace, mockReader)
				Ω(c).Should(BeNil())
				Ω(err).Should(HaveOccurred())
				Ω(err.Error()).Should(ContainSubstring("could not read config file"))
			})

			It("should return an error if no pod template config is found", func() {
				mockReader.EXPECT().Get(ctx, cmKey, gm.AssignableToTypeOf(&corev1.ConfigMap{})).
					Do(func(ctx context.Context, key client.ObjectKey, cm *corev1.ConfigMap, opts ...client.GetOption) error {
						cm.Data = map[string]string{
							ConfigFileName: "name: foo",
						}
						return nil
					})

				c, err := getInternal(namespace, mockReader)
				Ω(c).Should(BeNil())
				Ω(err).Should(HaveOccurred())
				Ω(err.Error()).Should(ContainSubstring("could not find pod template"))
			})
		})

		Context("success", func() {
			It("should return a config without owner", func() {
				mockReader.EXPECT().Get(ctx, cmKey, gm.AssignableToTypeOf(&corev1.ConfigMap{})).
					Do(func(ctx context.Context, key client.ObjectKey, cm *corev1.ConfigMap, opts ...client.GetOption) error {
						cm.Data = map[string]string{
							ConfigFileName:  "name: foo",
							PodTemplateName: "kind: Pod",
						}
						return nil
					})
				mockReader.EXPECT().Get(ctx, gm.Any(), gm.AssignableToTypeOf(&corev1.Pod{})).
					Return(fmt.Errorf("pod not found"))

				c, err := getInternal(namespace, mockReader)
				Ω(c).ShouldNot(BeNil())
				Ω(err).Should(BeNil())

				Ω(c.JobPodTemplate).Should(Equal("kind: Pod"))
				Ω(c.Owner).Should(BeNil())
			})

			It("should return a config with owner", func() {
				mockReader.EXPECT().Get(ctx, cmKey, gm.AssignableToTypeOf(&corev1.ConfigMap{})).
					Do(func(ctx context.Context, key client.ObjectKey, cm *corev1.ConfigMap, opts ...client.GetOption) error {
						cm.Data = map[string]string{
							ConfigFileName:  "name: foo",
							PodTemplateName: "kind: Pod",
						}
						return nil
					})
				mockReader.EXPECT().Get(ctx, gm.Any(), gm.AssignableToTypeOf(&corev1.Pod{})).
					Do(func(ctx context.Context, key client.ObjectKey, pod *corev1.Pod, opts ...client.GetOption) error {
						pod.OwnerReferences = []metav1.OwnerReference{
							{
								Kind: "ReplicaSet",
								Name: "rs-1",
							},
						}
						return nil
					})
				mockReader.EXPECT().Get(ctx, gm.Any(), gm.AssignableToTypeOf(&unstructured.Unstructured{})).
					Do(func(ctx context.Context, key client.ObjectKey, us *unstructured.Unstructured, opts ...client.GetOption) error {
						us.Object["metadata"] = map[string]interface{}{
							"ownerReferences": []interface{}{
								map[string]interface{}{
									"kind": "Deployment",
									"name": "deployment-1",
								},
							},
						}
						return nil
					})
				mockReader.EXPECT().Get(ctx, gm.Any(), gm.AssignableToTypeOf(&unstructured.Unstructured{})).
					Do(func(ctx context.Context, key client.ObjectKey, us *unstructured.Unstructured, opts ...client.GetOption) error {
						us.Object["metadata"] = map[string]interface{}{
							"name": "deployment-1",
						}
						us.Object["kind"] = "Deployment"
						return nil
					})
				c, err := getInternal(namespace, mockReader)
				Ω(c).ShouldNot(BeNil())
				Ω(err).Should(BeNil())

				Ω(c.JobPodTemplate).Should(Equal("kind: Pod"))
				Ω(c.Owner).ShouldNot(BeNil())
				Ω(c.Owner.GetObjectKind().GroupVersionKind().Kind).Should(Equal("Deployment"))
				Ω(c.Owner).Should(
					WithTransform(func(o runtime.Object) string {
						return o.(metav1.Object).GetName()
					}, Equal("deployment-1")))
			})
		})
	})
})
