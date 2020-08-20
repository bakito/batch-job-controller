package config_test

import (
	"context"
	"fmt"
	"os"

	"github.com/bakito/batch-job-controller/pkg/config"
	mock_client "github.com/bakito/batch-job-controller/pkg/mocks/client"
	gm "github.com/golang/mock/gomock"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Config", func() {
	Context("Metrics", func() {
		var (
			m *config.Metrics
		)
		BeforeEach(func() {
			m = &config.Metrics{
				Prefix: "my_metric",
			}
		})
		It("should return a correct name", func() {
			Ω(m.NameFor("name")).To(Equal("my_metric_name"))
		})
	})

	Context("Get", func() {
		var (
			ctx      context.Context
			mockCtrl *gm.Controller //gomock struct
			//			mockLog    *mock_logr.MockLogger
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
			//			mockLog = mock_logr.NewMockLogger(mockCtrl)
			mockReader = mock_client.NewMockReader(mockCtrl)
			_ = os.Setenv(config.EnvConfigMapName, cmName)
			_ = os.Setenv(config.EnvHostname, podName)
			cmKey = client.ObjectKey{Namespace: namespace, Name: cmName}
		})

		Context("error", func() {
			It("should return an error returned by the reader", func() {
				mockReader.EXPECT().Get(ctx, cmKey, gm.AssignableToTypeOf(&corev1.ConfigMap{})).
					Return(fmt.Errorf("error"))

				c, err := config.Get(namespace, mockReader)
				Ω(c).To(BeNil())
				Ω(err).To(HaveOccurred())
				Ω(err.Error()).To(ContainSubstring("error getting configmap"))
			})

			It("should return an error if no config is found", func() {
				mockReader.EXPECT().Get(ctx, cmKey, gm.AssignableToTypeOf(&corev1.ConfigMap{})).
					Do(func(ctx context.Context, key client.ObjectKey, cm *corev1.ConfigMap) error {
						cm.Data = map[string]string{}
						return nil
					})

				c, err := config.Get(namespace, mockReader)
				Ω(c).To(BeNil())
				Ω(err).To(HaveOccurred())
				Ω(err.Error()).To(ContainSubstring("could not find config file"))
			})

			It("should return an error if no config can not be parsed", func() {
				mockReader.EXPECT().Get(ctx, cmKey, gm.AssignableToTypeOf(&corev1.ConfigMap{})).
					Do(func(ctx context.Context, key client.ObjectKey, cm *corev1.ConfigMap) error {
						cm.Data = map[string]string{
							config.ConfigFileName: "foo",
						}
						return nil
					})

				c, err := config.Get(namespace, mockReader)
				Ω(c).To(BeNil())
				Ω(err).To(HaveOccurred())
				Ω(err.Error()).To(ContainSubstring("could not read config file"))
			})

			It("should return an error if no pod template config is found", func() {
				mockReader.EXPECT().Get(ctx, cmKey, gm.AssignableToTypeOf(&corev1.ConfigMap{})).
					Do(func(ctx context.Context, key client.ObjectKey, cm *corev1.ConfigMap) error {
						cm.Data = map[string]string{
							config.ConfigFileName: "name: foo",
						}
						return nil
					})

				c, err := config.Get(namespace, mockReader)
				Ω(c).To(BeNil())
				Ω(err).To(HaveOccurred())
				Ω(err.Error()).To(ContainSubstring("could not find pod template"))
			})
		})

		Context("success", func() {
			It("should return a config without owner", func() {

				mockReader.EXPECT().Get(ctx, cmKey, gm.AssignableToTypeOf(&corev1.ConfigMap{})).
					Do(func(ctx context.Context, key client.ObjectKey, cm *corev1.ConfigMap) error {
						cm.Data = map[string]string{
							config.ConfigFileName:  "name: foo",
							config.PodTemplateName: "kind: Pod",
						}
						return nil
					})
				mockReader.EXPECT().Get(ctx, gm.Any(), gm.AssignableToTypeOf(&corev1.Pod{})).
					Return(fmt.Errorf("pod not found"))

				c, err := config.Get(namespace, mockReader)
				Ω(c).ToNot(BeNil())
				Ω(err).To(BeNil())

				Ω(c.JobPodTemplate).To(Equal("kind: Pod"))
				Ω(c.Owner).To(BeNil())
			})

			It("should return a config with owner", func() {

				mockReader.EXPECT().Get(ctx, cmKey, gm.AssignableToTypeOf(&corev1.ConfigMap{})).
					Do(func(ctx context.Context, key client.ObjectKey, cm *corev1.ConfigMap) error {
						cm.Data = map[string]string{
							config.ConfigFileName:  "name: foo",
							config.PodTemplateName: "kind: Pod",
						}
						return nil
					})
				mockReader.EXPECT().Get(ctx, gm.Any(), gm.AssignableToTypeOf(&corev1.Pod{})).
					Do(func(ctx context.Context, key client.ObjectKey, pod *corev1.Pod) error {
						pod.OwnerReferences = []metav1.OwnerReference{
							{
								Kind: "ReplicaSet",
								Name: "rs-1",
							},
						}
						return nil
					})
				mockReader.EXPECT().Get(ctx, gm.Any(), gm.AssignableToTypeOf(&unstructured.Unstructured{})).
					Do(func(ctx context.Context, key client.ObjectKey, us *unstructured.Unstructured) error {
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
					Do(func(ctx context.Context, key client.ObjectKey, us *unstructured.Unstructured) error {
						us.Object["metadata"] = map[string]interface{}{
							"name": "deployment-1",
						}
						us.Object["kind"] = "Deployment"
						return nil
					})
				c, err := config.Get(namespace, mockReader)
				Ω(c).ToNot(BeNil())
				Ω(err).To(BeNil())

				Ω(c.JobPodTemplate).To(Equal("kind: Pod"))
				Ω(c.Owner).ToNot(BeNil())
				Ω(c.Owner.GetObjectKind().GroupVersionKind().Kind).To(Equal("Deployment"))
				Ω(c.Owner).To(
					WithTransform(func(o runtime.Object) string {
						return o.(metav1.Object).GetName()
					}, Equal("deployment-1")))
			})
		})
	})
})
