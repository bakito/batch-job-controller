package lifecycle_test

import (
	"github.com/bakito/batch-job-controller/pkg/config"
	"github.com/bakito/batch-job-controller/pkg/lifecycle"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("types", func() {
	Context("Results.Validate", func() {
		var (
			results lifecycle.Results
			cfg     *config.Config
		)
		BeforeEach(func() {
			results = lifecycle.Results{
				"aaa": []lifecycle.Result{},
			}
			cfg = &config.Config{
				Metrics: config.Metrics{
					Prefix: "foo",
				},
			}
		})
		It("should be valid", func() {
			err := results.Validate(cfg)
			Ω(err).ShouldNot(HaveOccurred())
		})
		It("should be invalid if prefix is not a valid prometheus metric name", func() {
			results["b b b"] = []lifecycle.Result{}
			err := results.Validate(cfg)
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(HaveSuffix("is not a valid metric name"))
		})
		It("should be invalid if results is empty", func() {
			delete(results, "aaa")
			err := results.Validate(cfg)
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(Equal("results must not be empty"))
		})
	})
})
