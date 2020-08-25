package metrics

import (
	"testing"

	"github.com/bakito/batch-job-controller/pkg/config"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	metricPrefix      string
	customGaugeName   string
	customGaugeHelp   string
	customGaugeLabel1 string
	customGaugeLabel2 string
	cfgMetrics        config.Metrics
)

func TestLifecycle(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Metrics Suite")
}

var _ = BeforeSuite(func() {
	metricPrefix = puuid()
	customGaugeName = puuid()
	customGaugeHelp = uuid.New().String()
	customGaugeLabel1 = puuid()
	customGaugeLabel2 = puuid()
	cfgMetrics = config.Metrics{
		Prefix: metricPrefix,
		Gauges: map[string]config.Metric{
			customGaugeName: {
				Help:   customGaugeHelp,
				Labels: []string{customGaugeLabel1, customGaugeLabel2},
			},
		},
	}
})
