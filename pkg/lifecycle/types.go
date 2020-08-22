package lifecycle

import prom "github.com/prometheus/client_golang/prometheus"

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

func (r *Results) Validate() error {
	return nil
}

type customMetric struct {
	gauge  *prom.GaugeVec
	labels []string
}
