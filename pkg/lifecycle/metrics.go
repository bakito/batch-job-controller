package lifecycle

import (
	"fmt"
	"github.com/bakito/batch-job-controller/pkg/config"
	prom "github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
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
	namespace        string
	latestMetric     bool
}

// Describe returns all the descriptions of the collector
func (c *Collector) Describe(ch chan<- *prom.Desc) {
	c.executionIDGauge.Describe(ch)
	c.podsGauge.Describe(ch)

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
		g := c.gauges[name]
		if result.Labels == nil {
			result.Labels = make(map[string]string)
		}
		result.Labels[labelNode] = node
		result.Labels[labelExecutionId] = executionID

		var values []string
		for _, l := range g.labels {
			values = append(values, result.Labels[l])
		}

		g.gauge.withLabelValues(values...).Set(result.Value)
		if c.latestMetric {
			// replace the executionId with 'latest'
			values[len(values)-1] = labelValueLatest
			g.gauge.withLabelValues(values...).Set(result.Value)
		}
	}
}

func (c *Collector) processingError(node string, executionId string, err bool) {
	value := 0.
	if err {
		value = 1
	}
	c.procErrorGauge.withLabelValues(node, executionId).Set(value)
	if c.latestMetric {
		c.procErrorGauge.withLabelValues(node, labelValueLatest).Set(value)
	}
}

func (c *Collector) duration(node string, executionId string, d float64) {
	c.durationGauge.withLabelValues(node, executionId).Set(d)
	if c.latestMetric {
		c.durationGauge.withLabelValues(node, labelValueLatest).Set(d)
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
