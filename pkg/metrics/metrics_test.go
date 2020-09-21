package metrics

import (
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"

	"github.com/bakito/batch-job-controller/pkg/config"
	"github.com/bakito/batch-job-controller/version"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

var _ = Describe("metrics", func() {
	var (
		cfg *config.Config
	)
	BeforeEach(func() {
		cfg = &config.Config{
			Metrics: cfgMetrics,
		}
	})
	Context("NewPromCollector", func() {
		It("should be valid", func() {
			_, err := NewPromCollector(cfg)
			Ω(err).ShouldNot(HaveOccurred())
		})
	})
	Context("Metrics", func() {
		var (
			pc               *Collector
			res              Result
			node             string
			executionId      string
			executionIdValue float64
			v1               string
			v2               string
			metricValue      int
		)
		BeforeEach(func() {
			v1 = uuid.New().String()
			v2 = uuid.New().String()
			metricValue = rand.Int()

			node = uuid.New().String()
			i := rand.Int()
			executionIdValue = float64(i)
			executionId = strconv.Itoa(i)

			pc, _ = NewPromCollector(cfg)

			res = Result{
				Value: float64(metricValue),
				Labels: map[string]string{
					customGaugeLabel1: v1,
					customGaugeLabel2: v2,
				},
			}
		})

		It("check 'The current execution ID'", func() {

			pc.ExecutionStarted(executionIdValue)
			checkMetric(
				pc,
				currentExecutionHelp,
				fmt.Sprintf("%s_%s", cfg.Metrics.Prefix, currentExecutionMetric),
				map[string]string{},
				executionId,
			)
		})

		It("check success 'Node with processing error, 1: has error / 0: no error'", func() {
			pc.ProcessingFinished(node, executionId, false)
			checkMetric(
				pc,
				procErrorHelp,
				fmt.Sprintf("%s_%s", cfg.Metrics.Prefix, procErrorMetric),
				map[string]string{"executionID": executionId, "node": node},
				"0",
			)
		})

		It("check error 'Node with processing error, 1: has error / 0: no error'", func() {
			pc.ProcessingFinished(node, executionId, true)
			checkMetric(
				pc,
				procErrorHelp,
				fmt.Sprintf("%s_%s", cfg.Metrics.Prefix, procErrorMetric),
				map[string]string{"executionID": executionId, "node": node},
				"1",
			)
		})

		It("check error 'Execution Duration in milliseconds'", func() {
			d := rand.Int()
			duaration := float64(d)
			pc.Duration(node, executionId, duaration)
			checkMetric(
				pc,
				durationHelp,
				fmt.Sprintf("%s_%s", cfg.Metrics.Prefix, durationMetric),
				map[string]string{"executionID": executionId, "node": node},
				strconv.Itoa(d),
			)
		})

		It("check error 'The number of Pods started for the last execution'", func() {
			c := rand.Int()
			cnt := float64(c)
			pc.Pods(cnt)
			checkMetric(
				pc,
				podsHelp,
				fmt.Sprintf("%s_%s", cfg.Metrics.Prefix, podsMetric),
				map[string]string{},
				strconv.Itoa(c),
			)
		})

		It("check version", func() {
			c := rand.Int()
			cnt := float64(c)
			pc.Pods(cnt)
			checkMetric(
				pc,
				versionHelp,
				versionMetric,
				map[string]string{
					"name":          cfg.Name,
					"poolSize":      strconv.Itoa(cfg.PodPoolSize),
					"prefix":        cfg.Metrics.Prefix,
					"reportHistory": strconv.Itoa(cfg.ReportHistory),
					"version":       version.Version},
				"1",
			)
		})

		It("check dynamic metric", func() {
			pc.MetricFor(executionId, node, customGaugeName, res)
			checkMetric(
				pc,
				customGaugeHelp,
				cfg.Metrics.NameFor(customGaugeName),
				map[string]string{"executionID": executionId, "node": node, customGaugeLabel1: v1, customGaugeLabel2: v2},
				strconv.Itoa(metricValue),
			)
		})

		It("Prune", func() {
			pc.MetricFor(executionId, node, customGaugeName, res)
			pc.Prune(executionId)
			checkMissingMetric(
				pc,
				cfg.Metrics.NameFor(customGaugeName),
			)
		})
	})
})

func checkMissingMetric(collector *Collector, name string) {
	err := testutil.CollectAndCompare(collector, strings.NewReader(""), name)
	Ω(err).ShouldNot(HaveOccurred())
}

func checkMetric(collector *Collector, help string, name string, labels map[string]string, value string) {

	l := ""
	if len(labels) > 0 {
		var keys []string
		for k := range labels {
			keys = append(keys, k)
		}

		sort.Strings(keys)

		var labelValues []string
		for _, k := range keys {
			labelValues = append(labelValues, fmt.Sprintf(`%s="%s"`, k, labels[k]))
		}
		l = fmt.Sprintf("{%s}", strings.Join(labelValues, ","))
	}

	expected := fmt.Sprintf(`
		# HELP %s %s
		# TYPE %s gauge
		%s%s %s
	`, name, help, name, name, l, value)
	err := testutil.CollectAndCompare(collector, strings.NewReader(expected), name)
	Ω(err).ShouldNot(HaveOccurred())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz_")

// puuid returns a prom valid uuid
func puuid() string {
	b := make([]rune, 10)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}