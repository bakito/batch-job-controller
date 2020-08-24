package lifecycle

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"math/rand"
	"sort"
	"strconv"
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
			pc               *Collector
			res              Result
			node             string
			executionId      string
			executionIdValue float64
			gaugeName        string
			gaugeHelp        string
			l1               string
			l2               string
			v1               string
			v2               string
			metricValue      int
		)
		BeforeEach(func() {
			gaugeName = puuid()
			gaugeHelp = uuid.New().String()

			l1 = puuid()
			l2 = puuid()
			v1 = uuid.New().String()
			v2 = uuid.New().String()
			metricValue = rand.Int()

			node = uuid.New().String()
			i := rand.Int()
			executionIdValue = float64(i)
			executionId = strconv.Itoa(i)

			cfg.Metrics.Gauges = map[string]config.Metric{
				gaugeName: {
					Help:   gaugeHelp,
					Labels: []string{l1, l2},
				},
			}
			pc, _ = NewPromCollector(cfg)

			res = Result{
				Value: float64(metricValue),
				Labels: map[string]string{
					l1: v1,
					l2: v2,
				},
			}
		})

		It("check 'The current execution ID'", func() {

			pc.newExecution(executionIdValue)
			checkMetric(
				pc,
				"The current execution ID",
				fmt.Sprintf("%s_%s", cfg.Metrics.Prefix, currentExecutionMetric),
				map[string]string{},
				executionId,
			)
		})

		It("check success 'Node with processing error, 1: has error / 0: no error'", func() {
			pc.processingError(node, executionId, false)
			checkMetric(
				pc,
				"Node with processing error, 1: has error / 0: no error",
				fmt.Sprintf("%s_%s", cfg.Metrics.Prefix, procErrorMetric),
				map[string]string{"executionID": executionId, "node": node},
				"0",
			)
		})

		It("check error 'Node with processing error, 1: has error / 0: no error'", func() {
			pc.processingError(node, executionId, true)
			checkMetric(
				pc,
				"Node with processing error, 1: has error / 0: no error",
				fmt.Sprintf("%s_%s", cfg.Metrics.Prefix, procErrorMetric),
				map[string]string{"executionID": executionId, "node": node},
				"1",
			)
		})

		It("check error 'Execution duration in milliseconds'", func() {
			d := rand.Int()
			duaration := float64(d)
			pc.duration(node, executionId, duaration)
			checkMetric(
				pc,
				"Execution duration in milliseconds",
				fmt.Sprintf("%s_%s", cfg.Metrics.Prefix, durationMetric),
				map[string]string{"executionID": executionId, "node": node},
				strconv.Itoa(d),
			)
		})

		It("check error 'The number of pods started for the last execution'", func() {
			c := rand.Int()
			cnt := float64(c)
			pc.pods(cnt)
			checkMetric(
				pc,
				"The number of pods started for the last execution",
				fmt.Sprintf("%s_%s", cfg.Metrics.Prefix, podsMetric),
				map[string]string{},
				strconv.Itoa(c),
			)
		})

		It("check dynamic metric", func() {
			pc.metricFor(executionId, node, gaugeName, res)
			checkMetric(
				pc,
				gaugeHelp,
				cfg.Metrics.NameFor(gaugeName),
				map[string]string{"executionID": executionId, "node": node, l1: v1, l2: v2},
				strconv.Itoa(metricValue),
			)
		})

		It("prune", func() {
			pc.metricFor(executionId, node, gaugeName, res)
			pc.prune(executionId)
			checkMissingMetric(
				pc,
				cfg.Metrics.NameFor(gaugeName),
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
