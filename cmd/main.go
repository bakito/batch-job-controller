package cmd

import (
	"fmt"
	"os"
	"reflect"
	"strconv"

	bjcc "github.com/bakito/batch-job-controller/pkg/config"
	"github.com/bakito/batch-job-controller/pkg/controller"
	"github.com/bakito/batch-job-controller/pkg/cron"
	"github.com/bakito/batch-job-controller/pkg/inject"
	"github.com/bakito/batch-job-controller/pkg/job"
	"github.com/bakito/batch-job-controller/pkg/lifecycle"
	"github.com/bakito/batch-job-controller/pkg/metrics"
	"github.com/bakito/batch-job-controller/version"
	"github.com/go-logr/zapr"
	zap2 "go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	// EnvNamespace namespace env variable name
	EnvNamespace = "NAMESPACE"
)

var (
	scheme    = runtime.NewScheme()
	setupLog  = ctrl.Log.WithName("setup")
	namespace string
)

func init() {
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(appsv1.AddToScheme(scheme))
}

// Setup setup main
func Setup() *Main {
	SetupLogger(true)

	// read env variables
	if value, exists := os.LookupEnv(EnvNamespace); exists {
		namespace = value
	} else {
		setupLog.Error(nil, "missing environment variable", "name", EnvNamespace)
		os.Exit(1)
	}

	config := ctrl.GetConfigOrDie()

	cfg, err := bjcc.Get(namespace, config, scheme)
	if err != nil {
		setupLog.Error(err, "unable to get config")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(config, ctrl.Options{
		Scheme:                     scheme,
		MetricsBindAddress:         ":9153",
		LeaderElection:             !cfg.DevMode,
		LeaderElectionID:           cfg.Name + "-leader-election",
		LeaderElectionNamespace:    namespace,
		LeaderElectionResourceLock: cfg.LeaderElectionResourceLock,
		Namespace:                  namespace,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	setupLog.Info("starting",
		bjcc.LabelVersion, version.Version,
		bjcc.LabelName, cfg.Name,
		bjcc.LabelPoolSize, strconv.Itoa(cfg.PodPoolSize),
		bjcc.LabelReportHistory, strconv.Itoa(cfg.ReportHistory))

	pc, err := metrics.NewPromCollector(cfg)
	if err != nil {
		setupLog.Error(err, "error creating prometheus collector")
		os.Exit(1)
	}

	return &Main{
		Controller: lifecycle.NewController(cfg, pc),
		Config:     cfg,
		Manager:    mgr,
	}
}

func SetupLogger(json bool) {
	o := func(o *zap.Options) {
		o.DestWriter = os.Stderr
		o.Development = bjcc.IsDevMode()
		encCfg := zap2.NewProductionEncoderConfig()
		if json {
			o.Encoder = zapcore.NewJSONEncoder(encCfg)
		} else {
			o.Encoder = zapcore.NewConsoleEncoder(encCfg)
		}
	}

	ctrl.SetLogger(zapr.NewLogger(zap.NewRaw(o)))
	klog.SetLogger(ctrl.Log)
}

// Start start main
func (m *Main) Start(runnables ...manager.Runnable) {
	var envExtender []job.CustomPodEnv

	// setup runnables
	for _, r := range runnables {

		m.addToManager(r)

		if e, ok := r.(job.CustomPodEnv); ok {
			c := reflect.TypeOf(r)
			setupLog.WithValues("extender", c).Info("registering custom pod env extender")
			envExtender = append(envExtender, e)
		}
	}

	// setup cron job
	m.addToManager(cron.Job(envExtender...))

	// Setup a new controller to reconcile ReplicaSets
	setupLog.Info("Setting up controller")

	if err := (&controller.PodReconciler{
		Client:     m.Manager.GetClient(),
		Controller: m.Controller,
	}).SetupWithManager(m.Manager); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Pod")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := m.Manager.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func (m *Main) addToManager(r manager.Runnable) {
	if er, ok := r.(inject.EventRecorder); ok {
		if m.eventRecorder == nil {
			m.eventRecorder = m.Manager.GetEventRecorderFor(m.Config.Name)
		}
		er.InjectEventRecorder(m.eventRecorder)
	}

	if c, ok := r.(inject.Config); ok {
		c.InjectConfig(m.Config)
	}
	if c, ok := r.(inject.Controller); ok {
		c.InjectController(m.Controller)
	}
	if r, ok := r.(inject.Reader); ok {
		r.InjectReader(m.Manager.GetAPIReader())
	}

	_ = m.Manager.Add(r)
}

// CustomConfigValue get a custom config value
func (m *Main) CustomConfigValue(name string) interface{} {
	if v, ok := m.Config.Custom[name]; ok {
		return v
	}
	setupLog.Error(fmt.Errorf("custom config value %q must be defined", name), "missing custom config value")
	os.Exit(1)
	return nil
}

// CustomConfigString get a custom config value string
func (m *Main) CustomConfigString(name string) string {
	v := m.CustomConfigValue(name)
	if s, ok := v.(string); ok {
		return s
	}
	setupLog.Error(fmt.Errorf("custom config value %q must be a string", name), "wrong custom config value type")
	os.Exit(1)
	return ""
}

// Main struct
type Main struct {
	Config        *bjcc.Config
	Controller    lifecycle.Controller
	Manager       manager.Manager
	eventRecorder record.EventRecorder
}
