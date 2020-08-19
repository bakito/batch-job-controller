package config

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
)

type Config struct {
	Name                  string                 `json:"name"`
	Namespace             string                 `json:"-"`
	JobServiceAccount     string                 `json:"jobServiceAccount"`
	JobNodeSelector       map[string]string      `json:"jobNodeSelector"`
	RunOnUnscheduledNodes bool                   `json:"runOnUnscheduledNodes"`
	JobPodTemplate        string                 `json:"-"`
	CronExpression        string                 `json:"cronExpression"`
	ReportDirectory       string                 `json:"reportDirectory"`
	ReportHistory         int                    `json:"reportHistory"`
	PodPoolSize           int                    `json:"podPoolSize"`
	RunOnStartup          bool                   `json:"runOnStartup"`
	Metrics               Metrics                `json:"metrics"`
	Custom                map[string]interface{} `json:"custom"`
	CallbackServiceName   string                 `json:"callbackServiceName"`
	CallbackServicePort   int                    `json:"callbackServicePort"`
	Owner                 runtime.Object         `json:"-"`
}

type Metrics struct {
	Prefix string            `json:"prefix"`
	Gauges map[string]Metric `json:"gauges"`
}

func (m *Metrics) NameFor(name string) string {
	return fmt.Sprintf("%s_%s", m.Prefix, name)
}

type Metric struct {
	Help   string   `json:"help"`
	Labels []string `json:"labels"`
}
