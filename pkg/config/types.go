package config

import (
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
)

const (
	defaultHealthBindAddress         = ":9152"
	defaultMetricsBindAddressAddress = ":9153"
)

// Config struct
type Config struct {
	Name                  string            `json:"name"`
	JobServiceAccount     string            `json:"jobServiceAccount"`
	JobNodeSelector       map[string]string `json:"jobNodeSelector"`
	RunOnUnscheduledNodes bool              `json:"runOnUnscheduledNodes"`
	CronExpression        string            `json:"cronExpression"`
	ReportDirectory       string            `json:"reportDirectory"`
	ReportHistory         int               `json:"reportHistory"`
	PodPoolSize           int               `json:"podPoolSize"`
	RunOnStartup          bool              `json:"runOnStartup"`
	StartupDelay          time.Duration     `json:"startupDelay"`
	Metrics               Metrics           `json:"metrics"`
	HealthProbePort       int               `json:"healthProbePort"`
	// LatestMetricsLabel if true, each result metric is also created with executionID=latest
	LatestMetricsLabel bool                   `json:"latestMetricsLabel"`
	Custom             map[string]interface{} `json:"custom"`
	// CallbackServiceName if left blank, the pod IP is used for callback
	CallbackServiceName string `json:"callbackServiceName"`
	CallbackServicePort int    `json:"callbackServicePort"`
	// LeaderElectionResourceLock resource lock type. if empty default (resourcelock.ConfigMapsLeasesResourceLock) is used
	LeaderElectionResourceLock string `json:"leaderElectionResourceLock,omitempty"`

	Namespace      string         `json:"-"`
	JobPodTemplate string         `json:"-"`
	Owner          runtime.Object `json:"-"`
	DevMode        bool           `json:"-"`
}

// PodName get the name of the pod
func (cfg *Config) PodName(nodeName string, id string) string {
	nameParts := strings.Split(nodeName, ".")
	podName := fmt.Sprintf("%s-job-%s-%s", cfg.Name, nameParts[0], id)
	return podName
}

func (cfg *Config) HealthProbeBindAddress() string {
	if cfg.HealthProbePort == 0 {
		return defaultHealthBindAddress
	}
	return fmt.Sprintf(":%d", cfg.HealthProbePort)
}

// Metrics config
type Metrics struct {
	Port   int               `json:"port"`
	Prefix string            `json:"prefix"`
	Gauges map[string]Metric `json:"gauges"`
}

// NameFor get the name of a metric
func (m *Metrics) NameFor(name string) string {
	return fmt.Sprintf("%s_%s", m.Prefix, name)
}

func (m *Metrics) BindAddress() string {
	if m.Port == 0 {
		return defaultMetricsBindAddressAddress
	}
	return fmt.Sprintf(":%d", m.Port)
}

// Metric config
type Metric struct {
	Help   string   `json:"help"`
	Labels []string `json:"labels"`
}
