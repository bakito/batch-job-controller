package cmd

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	bjcc "github.com/bakito/batch-job-controller/pkg/config"
	"github.com/bakito/batch-job-controller/pkg/controller"
	"github.com/bakito/batch-job-controller/pkg/cron"
	"github.com/bakito/batch-job-controller/pkg/inject"
	"github.com/bakito/batch-job-controller/pkg/job"
	"github.com/bakito/batch-job-controller/pkg/lifecycle"
	"github.com/bakito/batch-job-controller/version"
	"github.com/go-logr/zapr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	// EnvNamespace namespace env variable name
	EnvNamespace = "NAMESPACE"
	// EnvDevMode enable dev mode
	EnvDevMode = "DEV_MODE"
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
	o := func(o *zap.Options) {
		o.DestWritter = os.Stderr
		o.Development = strings.ToLower(os.Getenv(EnvDevMode)) == "true"
	}

	ctrl.SetLogger(zapr.NewLogger(zap.NewRaw(o)))

	setupLog.Info("starting", "version", version.Version)

	// read env variables
	if value, exists := os.LookupEnv(EnvNamespace); exists {
		namespace = value
	} else {
		setupLog.Error(nil, "missing environment variable", "name", EnvNamespace)
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                  scheme,
		MetricsBindAddress:      ":9153",
		LeaderElection:          true,
		LeaderElectionID:        "9a62a63a.bakito.ch",
		LeaderElectionNamespace: namespace,
		Namespace:               namespace,
	})

	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	cfg, err := bjcc.Get(namespace, mgr.GetAPIReader())
	if err != nil {
		setupLog.Error(err, "unable to get config")
		os.Exit(1)
	}

	pc, err := lifecycle.NewPromCollector(namespace, cfg)
	if err != nil {
		setupLog.Error(err, "error creating prometheus collector")
		os.Exit(1)
	}
	cache := lifecycle.NewCache(cfg, pc)

	return &Main{
		Cache:   cache,
		Config:  cfg,
		Manager: mgr,
	}
}

// Start start main
func (m *Main) Start(runnables ...manager.Runnable) {

	var envExtender []job.CustomPodEnv

	var eventRecorder record.EventRecorder
	// setup runnables
	for _, r := range runnables {
		if er, ok := r.(inject.EventRecorder); ok {
			if eventRecorder == nil {
				eventRecorder = m.Manager.GetEventRecorderFor(m.Config.Name)
			}
			er.InjectEventRecorder(eventRecorder)
		}

		if c, ok := r.(inject.Config); ok {
			c.InjectConfig(m.Config)
		}
		if c, ok := r.(inject.Cache); ok {
			c.InjectCache(m.Cache)
		}
		if r, ok := r.(inject.Reader); ok {
			r.InjectReader(m.Manager.GetAPIReader())
		}

		_ = m.Manager.Add(r)
		if e, ok := r.(job.CustomPodEnv); ok {
			c := reflect.TypeOf(r)
			setupLog.WithValues("extender", c).Info("registering custom pod env extender")
			envExtender = append(envExtender, e)
		}
	}

	// setup cron job
	cj, err := cron.Job(namespace, m.Config, m.Manager.GetClient(), m.Cache, m.Config.Owner, envExtender...)
	if err != nil {
		setupLog.Error(err, "unable to set up cron job")
		os.Exit(1)
	}

	cj.Start()

	// Setup a new controller to reconcile ReplicaSets
	setupLog.Info("Setting up controller")

	if err = (&controller.PodReconciler{
		Client: m.Manager.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Pod"),
		Cache:  m.Cache,
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
	Config  *bjcc.Config
	Cache   lifecycle.Cache
	Manager manager.Manager
}
