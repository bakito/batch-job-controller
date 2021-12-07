package metrics

import (
	"fmt"
	"sync"

	"github.com/bakito/batch-job-controller/pkg/config"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
)

// Result metrics result
type Result struct {
	Value  float64           `json:"value"`
	Labels map[string]string `json:"labels,omitempty"`
}

// Results map of results
type Results map[string][]Result

// Validate the results
func (r Results) Validate(cfg *config.Config) error {
	if len(r) == 0 {
		return fmt.Errorf("results must not be empty")
	}
	for name := range r {
		if !model.IsValidMetricName(model.LabelValue(cfg.Metrics.NameFor(name))) {
			return fmt.Errorf("%q is not a valid metric name", name)
		}
	}
	return nil
}

type customMetric struct {
	gauge  *executionIDMetric
	labels []string
}

func newMetric(opts prom.GaugeOpts, labelNames ...string) *executionIDMetric {
	return &executionIDMetric{
		gauge:  prom.NewGaugeVec(opts, labelNames),
		labels: map[string][][]string{},
	}
}

type executionIDMetric struct {
	gauge  *prom.GaugeVec
	labels map[string][][]string
	mux    sync.Mutex
}

func (m *executionIDMetric) describe(ch chan<- *prom.Desc) {
	m.gauge.Describe(ch)
}

func (m *executionIDMetric) collect(ch chan<- prom.Metric) {
	m.gauge.Collect(ch)
}

func (m *executionIDMetric) prune(executionID string) {
	if labelSets, ok := m.labels[executionID]; ok {
		for _, labelSet := range labelSets {
			m.gauge.DeleteLabelValues(labelSet...)
		}
	}
}

func (m *executionIDMetric) withLabelValues(labels ...string) prom.Gauge {
	exID := labels[len(labels)-1]
	m.cacheLabels(exID, labels)
	return m.gauge.WithLabelValues(labels...)
}

func (m *executionIDMetric) cacheLabels(exID string, labels []string) {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.labels[exID] = append(m.labels[exID], labels)
}
