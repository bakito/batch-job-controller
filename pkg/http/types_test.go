package http_test

import (
	"github.com/bakito/batch-job-controller/pkg/http"
	. "github.com/onsi/ginkgo"
)

var _ = Describe("types", func() {
	var (
		event *http.Event
	)
	BeforeEach(func() {
		event = &http.Event{}
	})
	Context("Event.Validate", func() {
		It("shoud be valid", func() {
			event.Validate()
		})
	})
})
