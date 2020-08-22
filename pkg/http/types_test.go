package http_test

import (
	"github.com/bakito/batch-job-controller/pkg/http"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("types", func() {
	var (
		event *http.Event
	)
	BeforeEach(func() {
		event = &http.Event{
			Eventtype: "Normal",
			Reason:    "AnyReason",
			Message:   "message",
		}
	})
	Context("Event.Validate", func() {
		It("should be valid", func() {
			err := event.Validate()
			Ω(err).ShouldNot(HaveOccurred())
		})
		It("should fail with an invalid eventtype", func() {
			event.Eventtype = "foo"
			err := event.Validate()
			Ω(err).Should(HaveOccurred())
		})
		It("should fail with an empty eventtype", func() {
			event.Eventtype = ""
			err := event.Validate()
			Ω(err).Should(HaveOccurred())
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
		It("should fail with with both messages empty", func() {
			event.Message = ""
			event.MessageFmt = ""
			err := event.Validate()
			Ω(err).Should(HaveOccurred())
		})
		It("should be valid with messages", func() {
			event.Message = "foo"
			event.MessageFmt = ""
			err := event.Validate()
			Ω(err).ShouldNot(HaveOccurred())
		})
		It("should be valid with messagesFmt", func() {
			event.Message = ""
			event.MessageFmt = "foo %s"
			event.Args = []string{"bar"}
			err := event.Validate()
			Ω(err).ShouldNot(HaveOccurred())
		})
	})
})
