package lifecycle

import prom "github.com/prometheus/client_golang/prometheus"

type ExecutionIDNotFound error

type Result struct {
	Value  float64           `json:"value"`
	Labels map[string]string `json:"labels"`
}

type customMetric struct {
	gauge  *prom.GaugeVec
	labels []string
}
