package metrics

import (
	"fmt"
	"strconv"

	"github.com/bakito/batch-job-controller/pkg/config"
	"github.com/bakito/batch-job-controller/version"
	prom "github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	labelNode        = "node"
	labelValueLatest = "latest"
	labelExecutionId = "executionID"
	labelPrefix      = "prefix"

	versionMetric = "com_github_bakito_batch_job_controller"

	procErrorHelp = "Node with processing error, 1: has error / 0: no error"
	versionHelp   = "information about github.com/bakito/batch-job-controller"
	podsHelp      = "The number of pods started for the last execution"

	currentExecutionHelp = "The current execution ID"
	durationHelp         = "Execution Duration in milliseconds"

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

func (c *Collector) ExecutionStarted(executionId float64) {
	c.executionIDGauge.WithLabelValues().Set(executionId)
}

// Prune metrics assigned to the given execution ID
func (c *Collector) Prune(executionId string) {
	c.procErrorGauge.prune(executionId)
	c.durationGauge.prune(executionId)
	for k := range c.gauges {
		c.gauges[k].gauge.prune(executionId)
	}
}

func (c *Collector) MetricFor(executionID string, node string, name string, result Result) {
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
			if len(values) > 0 {
				// replace the executionId with 'latest'
				values[len(values)-1] = labelValueLatest
			}
			g.gauge.withLabelValues(values...).Set(result.Value)
		}
	}
}

func (c *Collector) ProcessingFinished(node string, executionId string, err bool) {
	value := 0.
	if err {
		value = 1
	}
	c.procErrorGauge.withLabelValues(node, executionId).Set(value)
	if c.latestMetric {
		c.procErrorGauge.withLabelValues(node, labelValueLatest).Set(value)
	}
}

func (c *Collector) Duration(node string, executionId string, d float64) {
	c.durationGauge.withLabelValues(node, executionId).Set(d)
	if c.latestMetric {
		c.durationGauge.withLabelValues(node, labelValueLatest).Set(d)
	}
}

func (c *Collector) Pods(cnt float64) {
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
		Help: currentExecutionHelp,
	}, []string{})

	c.procErrorGauge = newMetric(prom.GaugeOpts{
		Name: fmt.Sprintf("%s_%s", cfg.Metrics.Prefix, procErrorMetric),
		Help: procErrorHelp,
	}, labelNode, labelExecutionId)

	c.durationGauge = newMetric(prom.GaugeOpts{
		Name: fmt.Sprintf("%s_%s", cfg.Metrics.Prefix, durationMetric),
		Help: durationHelp,
	}, labelNode, labelExecutionId)

	c.podsGauge = prom.NewGaugeVec(prom.GaugeOpts{
		Name: fmt.Sprintf("%s_%s", cfg.Metrics.Prefix, podsMetric),
		Help: podsHelp,
	}, []string{})

	c.versionGauge = prom.NewGaugeVec(prom.GaugeOpts{
		Name: versionMetric,
		Help: versionHelp,
	}, []string{config.LabelVersion, config.LabelName, labelPrefix, config.LabelPoolSize, config.LabelReportHistory})

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

	c.versionGauge.WithLabelValues(
		version.Version,
		cfg.Name,
		cfg.Metrics.Prefix,
		strconv.Itoa(cfg.PodPoolSize),
		strconv.Itoa(cfg.ReportHistory),
	).Set(1)

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
