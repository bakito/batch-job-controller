package lifecycle

import (
	"fmt"
	"github.com/bakito/batch-job-controller/version"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"strconv"

	"github.com/bakito/batch-job-controller/pkg/config"
	prom "github.com/prometheus/client_golang/prometheus"
)

const (
	labelNode        = "node"
	labelValueLatest = "latest"
	labelExecutionId = "executionID"
)

var (
	currentExecutionMetric = "current_execution_id"
	procErrorMetric        = "processing"
	durationMetric         = "duration"
	podsMetric             = "pods"
)

// Collector struct
type Collector struct {
	gauges           map[string]customMetric
	executionIDGauge *prom.GaugeVec
	procErrorGauge   *executionIDMetric
	durationGauge    *executionIDMetric
	podsGauge        *prom.GaugeVec
	versionGauge     *prom.GaugeVec
	namespace        string
	latestMetric     bool
}

// Describe returns all the descriptions of the collector
func (c *Collector) Describe(ch chan<- *prom.Desc) {
	c.executionIDGauge.Describe(ch)
	c.podsGauge.Describe(ch)
	c.versionGauge.Describe(ch)

	c.procErrorGauge.describe(ch)
	c.durationGauge.describe(ch)
	for k := range c.gauges {
		c.gauges[k].gauge.describe(ch)
	}
}

// collect returns the current state of the metrics
func (c *Collector) Collect(ch chan<- prom.Metric) {
	c.executionIDGauge.Collect(ch)
	c.podsGauge.Collect(ch)
	c.versionGauge.Collect(ch)

	c.procErrorGauge.collect(ch)
	c.durationGauge.collect(ch)
	for k := range c.gauges {
		c.gauges[k].gauge.collect(ch)
	}
}

func (c *Collector) newExecution(executionId float64) {
	c.executionIDGauge.WithLabelValues().Set(executionId)
}

// prune metrics assigned to the given execution ID
func (c *Collector) prune(executionId string) {
	c.procErrorGauge.prune(executionId)
	c.durationGauge.prune(executionId)
	for k := range c.gauges {
		c.gauges[k].gauge.prune(executionId)
	}
}

func (c *Collector) metricFor(executionID string, node string, name string, result Result) {
	if _, ok := c.gauges[name]; ok {
		if result.Labels == nil {
			result.Labels = make(map[string]string)
		}
		result.Labels[labelNode] = node
		result.Labels[labelExecutionId] = executionID

		var labels []string
		for _, l := range c.gauges[name].labels {
			labels = append(labels, result.Labels[l])
		}

		c.gauges[name].gauge.withLabelValues(labels...).Set(result.Value)
		if c.latestMetric {
			// replace the executionId with 'latest'
			labels[len(labels)-1] = labelValueLatest
			c.gauges[name].gauge.withLabelValues(labels...).Set(result.Value)
		}
	}
}

func (c *Collector) processingError(name string, executionId string, err bool) {
	value := 0.
	if err {
		value = 1
	}
	c.procErrorGauge.withLabelValues(name, executionId).Set(value)
	if c.latestMetric {
		c.procErrorGauge.withLabelValues(name, labelValueLatest).Set(value)
	}
}

func (c *Collector) duration(name string, executionId string, d float64) {
	c.durationGauge.withLabelValues(name, executionId).Set(d)
	if c.latestMetric {
		c.durationGauge.withLabelValues(name, labelValueLatest).Set(d)
	}
}

func (c *Collector) pods(cnt float64) {
	g, err := c.podsGauge.GetMetricWithLabelValues()
	if err == nil {
		g.Set(cnt)
	}
}

// NewPromCollector create a new prom collector
func NewPromCollector(cfg *config.Config) (*Collector, error) {

	c := &Collector{
		gauges:       make(map[string]customMetric),
		namespace:    cfg.Namespace,
		latestMetric: cfg.LatestMetricsLabel,
	}
	c.executionIDGauge = prom.NewGaugeVec(prom.GaugeOpts{
		Name: fmt.Sprintf("%s_%s", cfg.Metrics.Prefix, currentExecutionMetric),
		Help: "The current execution ID",
	}, []string{})

	c.procErrorGauge = newMetric(prom.GaugeOpts{
		Name: fmt.Sprintf("%s_%s", cfg.Metrics.Prefix, procErrorMetric),
		Help: "Node with processing error, 1: has error / 0: no error",
	}, labelNode, labelExecutionId)

	c.durationGauge = newMetric(prom.GaugeOpts{
		Name: fmt.Sprintf("%s_%s", cfg.Metrics.Prefix, durationMetric),
		Help: "execution duration in milliseconds",
	}, labelNode, labelExecutionId)

	c.podsGauge = prom.NewGaugeVec(prom.GaugeOpts{
		Name: fmt.Sprintf("%s_%s", cfg.Metrics.Prefix, podsMetric),
		Help: "the number of pods started for the last execution",
	}, []string{})

	c.versionGauge = prom.NewGaugeVec(prom.GaugeOpts{
		Name: "com_github_bakito_batch_job_controller",
		Help: "information about github.com/bakito/batch-job-controller",
	}, []string{config.LabelVersion, config.LabelName, config.LabelPoolSize, config.LabelReportHistory})

	for name, metric := range cfg.Metrics.Gauges {
		if name == procErrorMetric || name == durationMetric || name == podsMetric {
			return nil, fmt.Errorf("the metric name %q is not allowed, it's one of the reserved names: %v",
				name, []string{procErrorMetric, durationMetric, podsMetric})
		}

		labels := enrichLabels(metric.Labels)

		c.gauges[name] = customMetric{
			labels: labels,
			gauge: newMetric(prom.GaugeOpts{
				Name: cfg.Metrics.NameFor(name),
				Help: metric.Help,
			}, labels...),
		}

	}

	metrics.Registry.Unregister(c)
	metrics.Registry.MustRegister(c)
	c.versionGauge.WithLabelValues(version.Version, cfg.Name, strconv.Itoa(cfg.PodPoolSize), strconv.Itoa(cfg.ReportHistory)).Set(1)
	return c, nil
}

func enrichLabels(labels []string) []string {
	out := labels
	m := make(map[string]bool)
	for _, l := range labels {
		m[l] = true
	}

	if _, ok := m[labelNode]; !ok {
		out = append(out, labelNode)
	}
	if _, ok := m[labelExecutionId]; !ok {
		out = append(out, labelExecutionId)
	}

	return out
}
