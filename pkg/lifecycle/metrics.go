package lifecycle

import (
	"fmt"

	"github.com/bakito/batch-job-controller/pkg/config"
	prom "github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	labelNode        = "node"
	labelExecutionId = "executionID"
)

var (
	procErrorMetric = "processing"
	durationMetric  = "duration"
	podsMetric      = "pods"
)

// Collector strunct
type Collector struct {
	gauges         map[string]customMetric
	procErrorGauge *prom.GaugeVec
	durationGauge  *prom.GaugeVec
	podsGauge      *prom.GaugeVec
	namespace      string
}

// Describe returns all the descriptions of the collector
func (c *Collector) Describe(ch chan<- *prom.Desc) {
	c.procErrorGauge.Describe(ch)
	c.durationGauge.Describe(ch)
	c.podsGauge.Describe(ch)
	for k := range c.gauges {
		c.gauges[k].gauge.Describe(ch)
	}
}

// Collect returns the current state of the metrics
func (c *Collector) Collect(ch chan<- prom.Metric) {
	c.procErrorGauge.Collect(ch)
	c.durationGauge.Collect(ch)
	c.podsGauge.Collect(ch)
	for k := range c.gauges {
		c.gauges[k].gauge.Collect(ch)
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
		c.gauges[name].gauge.WithLabelValues(labels...).Set(result.Value)
	}
}

func (c *Collector) processingError(name string, executionId string, err bool) {
	value := 0.
	if err {
		value = 1
	}
	c.procErrorGauge.WithLabelValues(name, executionId).Set(value)
}

func (c *Collector) duration(name string, executionId string, d float64) {
	c.durationGauge.WithLabelValues(name, executionId).Set(d)
}

func (c *Collector) pods(cnt float64) {
	g, err := c.podsGauge.GetMetricWithLabelValues()
	if err == nil {
		g.Set(cnt)
	}
}

// NewPromCollector create a new prom collector
func NewPromCollector(namespace string, cfg *config.Config) (*Collector, error) {

	c := &Collector{
		gauges:    make(map[string]customMetric),
		namespace: namespace,
	}
	c.procErrorGauge = prom.NewGaugeVec(prom.GaugeOpts{
		Name: fmt.Sprintf("%s_%s", cfg.Metrics.Prefix, procErrorMetric),
		Help: "Node with processing error, 1: has error / 0: no error",
	}, []string{labelNode, labelExecutionId})

	c.durationGauge =
		prom.NewGaugeVec(prom.GaugeOpts{
			Name: fmt.Sprintf("%s_%s", cfg.Metrics.Prefix, durationMetric),
			Help: "execution duration in milliseconds",
		}, []string{labelNode, labelExecutionId})

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
			gauge: prom.NewGaugeVec(prom.GaugeOpts{
				Name: cfg.Metrics.NameFor(name),
				Help: metric.Help,
			}, labels),
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
