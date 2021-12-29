package cron

import (
	"context"
	"os"
	"time"

	"github.com/bakito/batch-job-controller/pkg/config"
	"github.com/bakito/batch-job-controller/pkg/job"
	"github.com/bakito/batch-job-controller/pkg/lifecycle"
	"github.com/go-logr/logr"
	"github.com/robfig/cron/v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var log = ctrl.Log.WithName("cron")

// Job creates a new Job runner instance
func Job(extender ...job.CustomPodEnv) manager.Runnable {
	return &cronJob{
		extender: extender,
	}
}

type cronJob struct {
	client     client.Client
	controller lifecycle.Controller
	running    bool
	cfg        *config.Config
	extender   []job.CustomPodEnv
}

// InjectConfig inject the config
func (j *cronJob) InjectConfig(cfg *config.Config) {
	j.cfg = cfg
}

// InjectController inject the controller
func (j *cronJob) InjectController(c lifecycle.Controller) {
	j.controller = c
}

// InjectClient inject the client
func (j *cronJob) InjectClient(c client.Client) error {
	j.client = c
	return nil
}

// NeedLeaderElection may only start if leader is elected
func (j *cronJob) NeedLeaderElection() bool {
	return true
}

// Start implement manager.Runnable
func (j *cronJob) Start(_ context.Context) error {
	log.WithValues("expression", j.cfg.CronExpression).Info("starting cron")
	c := cron.New()
	_, err := c.AddFunc(j.cfg.CronExpression, j.startPods)
	if err != nil {
		return err
	}

	if j.cfg.RunOnStartup {
		go func() {
			log.WithValues("delay", j.cfg.StartupDelay).Info("starting on startup")
			time.Sleep(j.cfg.StartupDelay)
			log.Info("starting")
			j.startPods()
		}()
	}

	c.Start()
	return nil
}

func (j *cronJob) deleteAll(obj client.Object) error {
	return j.client.DeleteAllOf(
		context.TODO(),
		obj,
		client.InNamespace(j.cfg.Namespace),
		job.MatchingLabels(j.cfg.Name),
		client.PropagationPolicy(metav1.DeletePropagationBackground),
	) // set propagation policy to also delete assigned pods
}

func (j *cronJob) startPods() {
	if j.running {
		log.Info("last cronjob still running")
		return
	}
	j.running = true
	defer func() {
		j.running = false
	}()

	// Fetch the ReplicaSet from the controller
	nodeList := &corev1.NodeList{}
	err := j.client.List(context.TODO(), nodeList, client.MatchingLabels(j.cfg.JobNodeSelector))
	if err != nil {
		log.Error(err, "error listing nodes")
		return
	}
	var nodes []corev1.Node
	for _, n := range nodeList.Items {
		if isUsable(n, j.cfg.RunOnUnscheduledNodes) {
			nodes = append(nodes, n)
		}
	}

	executionID := j.controller.NewExecution(len(nodes))

	jobLog := log.WithValues("id", executionID)

	jobLog.Info("deleting old job pods")
	err = j.deleteAll(&corev1.Pod{})
	if err != nil {
		jobLog.Error(err, "unable to delete old pods")
		return
	}

	var callbackAddress string
	if ip, ok := os.LookupEnv(config.EnvPodIP); ok {
		callbackAddress = ip
	} else {
		// get service
		svc := &corev1.Service{}
		err = j.client.Get(context.TODO(), client.ObjectKey{Namespace: j.cfg.Namespace, Name: j.cfg.CallbackServiceName}, svc)
		if err != nil {
			jobLog.WithValues("service-name", j.cfg.CallbackServiceName).Error(err, "error getting service")
			return
		}
		callbackAddress = svc.Spec.ClusterIP
	}

	jobLog.Info("executing job")
	for _, n := range nodes {
		pod, err := job.New(j.cfg, n.ObjectMeta.Name, executionID, callbackAddress, j.cfg.Owner, j.extender...)
		if err != nil {
			jobLog.Error(err, "error creating pod from template")
			return
		}

		_ = j.controller.AddPod(&podJob{
			id:       executionID,
			nodeName: n.ObjectMeta.Name,
			log:      jobLog,
			client:   j.client,
			pod:      pod,
		})
	}

	_ = j.controller.AllAdded(executionID)
}

func isUsable(node corev1.Node, runOnUnscheduledNodes bool) bool {
	if !runOnUnscheduledNodes && node.Spec.Unschedulable {
		return false
	}
	for _, c := range node.Status.Conditions {
		if c.Type == corev1.NodeReady {
			return c.Status == corev1.ConditionTrue
		}
	}
	return false
}

type podJob struct {
	id       string
	nodeName string
	log      logr.Logger
	pod      *corev1.Pod
	client   client.Client
}

func (j *podJob) ID() string {
	return j.id
}

func (j *podJob) Node() string {
	return j.nodeName
}

// CreatePod create a worker pod
func (j *podJob) CreatePod() {
	log.Info("create pod", "node", j.nodeName)
	err := j.client.Create(context.TODO(), j.pod)
	if err != nil {
		log.Error(err, "unable to create pod", "node", j.nodeName)
	}
}
