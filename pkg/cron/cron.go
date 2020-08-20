package cron

import (
	"context"
	ctrl "sigs.k8s.io/controller-runtime"
	"time"

	"github.com/bakito/batch-job-controller/pkg/config"
	"github.com/bakito/batch-job-controller/pkg/job"
	"github.com/bakito/batch-job-controller/pkg/lifecycle"
	"github.com/go-logr/logr"
	"github.com/robfig/cron/v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	log = ctrl.Log.WithName("cron")
)

//Job prepare the static file server
func Job(namespace string, cfg *config.Config, client client.Client, cache lifecycle.Cache, owner runtime.Object, extender ...job.CustomPodEnv) (*cron.Cron, error) {

	var cj = &cronJob{
		namespace: namespace,
		cache:     cache,
		cfg:       *cfg,
		client:    client,
		extender:  extender,
		owner:     owner,
	}
	log.WithValues("expression", cfg.CronExpression).Info("starting cron")

	c := cron.New()
	_, _ = c.AddFunc(cfg.CronExpression, cj.startPods)

	if cfg.RunOnStartup {
		go func() {
			time.Sleep(time.Second * 10)
			log.Info("starting cron on startup")
			cj.startPods()
		}()
	}

	return c, nil
}

type cronJob struct {
	namespace string
	client    client.Client
	job       *cron.Cron
	cache     lifecycle.Cache
	running   bool
	cfg       config.Config
	extender  []job.CustomPodEnv
	owner     runtime.Object
}

func (j *cronJob) deleteAll(obj runtime.Object) error {
	return j.client.DeleteAllOf(context.TODO(), obj,
		client.InNamespace(j.namespace),
		job.MatchingLabels(j.cfg.Name),
		client.PropagationPolicy(metav1.DeletePropagationBackground)) // set propagation policy to also delete assigned pods
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

	executionID := j.cache.NewExecution()

	jobLog := log.WithValues("id", executionID)

	err := j.deleteAll(&corev1.Pod{})
	if err != nil {
		jobLog.Error(err, "unable to delete old pods")
		return
	}

	// get service
	svc := &corev1.Service{}
	err = j.client.Get(context.TODO(), client.ObjectKey{Namespace: j.cfg.Namespace, Name: j.cfg.CallbackServiceName}, svc)
	if err != nil {
		jobLog.Error(err, "error getting service %q", j.cfg.CallbackServiceName)
	}

	// Fetch the ReplicaSet from the cache
	nodeList := &corev1.NodeList{}
	err = j.client.List(context.TODO(), nodeList, client.MatchingLabels(j.cfg.JobNodeSelector))
	if err != nil {
		jobLog.Error(err, "error listing nodes")
		return
	}

	jobLog.Info("executing job")
	for _, n := range nodeList.Items {
		if isUsable(n, j.cfg.RunOnUnscheduledNodes) {
			pod, err := job.New(j.cfg, n.ObjectMeta.Name, executionID, svc.Spec.ClusterIP, j.owner, j.extender...)
			if err != nil {
				jobLog.Error(err, "error creating pod from template")
				return
			}

			_ = j.cache.AddPod(&podJob{
				id:       executionID,
				nodeName: n.ObjectMeta.Name,
				log:      jobLog,
				client:   j.client,
				pod:      pod,
			})
		}
	}

	_ = j.cache.AllAdded(executionID)
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

func (j *podJob) Process() {
	log.Info("create pod", "node", j.nodeName)
	err := j.client.Create(context.TODO(), j.pod)
	if err != nil {
		log.Error(err, "unable to create pod", "node", j.nodeName)
	}
}
