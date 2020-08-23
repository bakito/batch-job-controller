package lifecycle

import (
	"fmt"
	"github.com/bakito/batch-job-controller/pkg/config"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
)

// ExecutionIDNotFound custom error
type ExecutionIDNotFound struct {
	Err error
}

func (e ExecutionIDNotFound) Error() string {
	return e.Err.Error()
}

// Result metrics result
type Result struct {
	Value  float64           `json:"value"`
	Labels map[string]string `json:"labels"`
}

type Results map[string][]Result

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
	gauge  *prom.GaugeVec
	labels []string
}
