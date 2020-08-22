package lifecycle_test

import (
	"github.com/bakito/batch-job-controller/pkg/config"
	"github.com/bakito/batch-job-controller/pkg/lifecycle"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("metrics", func() {
	Context("NewPromCollector", func() {
		var (
			cfg *config.Config
		)
		BeforeEach(func() {
			cfg = &config.Config{
				Metrics: config.Metrics{
					Prefix: "foo",
				},
			}
		})
		It("should be valid", func() {
			_, err := lifecycle.NewPromCollector("ns", cfg)
			Ω(err).ShouldNot(HaveOccurred())
		})
	})
})
