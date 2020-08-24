package lifecycle

import (
	"fmt"
	"github.com/bakito/batch-job-controller/version"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"math/rand"
	"strings"

	"github.com/bakito/batch-job-controller/pkg/config"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("metrics", func() {
	var (
		cfg          *config.Config
		metricPrefix string
	)
	BeforeEach(func() {
		metricPrefix = puuid()
		cfg = &config.Config{
			Metrics: config.Metrics{
				Prefix: metricPrefix,
			},
		}
	})
	Context("NewPromCollector", func() {
		It("should be valid", func() {
			_, err := NewPromCollector(cfg)
			Ω(err).ShouldNot(HaveOccurred())
		})
	})
	Context("NewPromCollector", func() {
		var (
			pc        *Collector
			res       Result
			node      string
			id        string
			value     string
			gaugeName string
			l1        string
			l2        string
		)
		BeforeEach(func() {
			gaugeName = puuid()

			l1 = puuid()
			l2 = puuid()

			node = uuid.New().String()
			value = uuid.New().String()
			id = uuid.New().String()

			cfg.Metrics.Gauges = map[string]config.Metric{
				gaugeName: {
					Help:   gaugeName,
					Labels: []string{l1, l2},
				},
			}
			pc, _ = NewPromCollector(cfg)

			res = Result{
				Labels: map[string]string{
					metricPrefix: value,
				},
			}
		})

		AfterEach(func() {
		})
		It("should be valid", func() {

			metadata := fmt.Sprintf(`
		# HELP com_github_bakito_batch_job_controller_%s information about github.com/bakito/batch-job-controller
		# TYPE com_github_bakito_batch_job_controller_%s gauge
	`, metricPrefix, metricPrefix)
			expected := fmt.Sprintf(`
		com_github_bakito_batch_job_controller_%s{name="%s",poolSize="%d",reportHistory="%d",version="%s"} 1
	`, metricPrefix, cfg.Name, cfg.PodPoolSize, cfg.ReportHistory, version.Version)

			pc.metricFor(id, node, metricPrefix, res)
			err := testutil.CollectAndCompare(pc, strings.NewReader(metadata+expected), fmt.Sprintf("com_github_bakito_batch_job_controller_%s", cfg.Metrics.Prefix))
			Ω(err).ShouldNot(HaveOccurred())
		})
	})
})

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz_")

// puuid returns a prom valid uuid
func puuid() string {
	b := make([]rune, 10)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
