package http_test

import (
	"github.com/bakito/batch-job-controller/pkg/http"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("types", func() {
	var event *http.Event
	BeforeEach(func() {
		event = &http.Event{
			Waring:  false,
			Reason:  "AnyReason",
			Message: "message",
		}
	})
	Context("Event.Validate", func() {
		It("should be valid", func() {
			err := event.Validate()
			Ω(err).ShouldNot(HaveOccurred())
		})
		It("should fail with an empty reason", func() {
			event.Reason = ""
			err := event.Validate()
			Ω(err).Should(HaveOccurred())
		})
		It("should fail with an lowercase reason", func() {
			event.Reason = "reason"
			err := event.Validate()
			Ω(err).Should(HaveOccurred())
		})
		It("should fail with with message empty", func() {
			event.Message = ""
			err := event.Validate()
			Ω(err).Should(HaveOccurred())
		})
	})
	Context("Event.Validate", func() {
		It("is Normal type", func() {
			event.Waring = false
			Ω(event.Type()).Should(Equal(corev1.EventTypeNormal))
		})
		It("is Warning type", func() {
			event.Waring = true
			Ω(event.Type()).Should(Equal(corev1.EventTypeWarning))
		})
	})
})
