package lifecycle

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bakito/batch-job-controller/pkg/config"
	"github.com/bakito/batch-job-controller/pkg/metrics"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

var log = ctrl.Log.WithName("lifecycle")

// NewController get a new controller
func NewController(cfg *config.Config, prom *metrics.Collector) Controller {
	return &controller{
		executions:    make(map[string]*execution),
		nodes:         make(map[string]bool),
		prom:          prom,
		log:           log.WithName("controller"),
		reportHistory: cfg.ReportHistory + 1, // 1+ for latest
		reportDir:     cfg.ReportDirectory,
		podPoolSize:   cfg.PodPoolSize,
		config:        *cfg,
	}
}

// Controller interface
type Controller interface {
	NewExecution(nbrOrJobs int) string
	AllAdded(executionID string) error
	AddPod(job Job) error
	PodTerminated(executionID, node string, phase corev1.PodPhase) error
	ReportReceived(executionID, node string, processingError error, results metrics.Results)
	Config() config.Config
	// Has return true if the executionId is known
	Has(node string, executionID string) bool
}

type controller struct {
	prom          *metrics.Collector
	executions    map[string]*execution
	nodes         map[string]bool
	log           logr.Logger
	reportDir     string
	reportHistory int
	podPoolSize   int
	config        config.Config
	progress      uint64
	progressStep  float64
}

// verify interface is implemented
var _ Controller = &controller{}

// Config get the config
func (c *controller) Config() config.Config {
	return c.config
}

func (c *controller) getProgress() string {
	return fmt.Sprintf("%.f%%", c.progressStep*float64(c.progress))
}

func (c *controller) addProgress(p uint64) {
	atomic.AddUint64(&c.progress, p)
}

// NewExecution setup a new execution
func (c *controller) NewExecution(jobs int) string {
	//                       yyyyMMddHHmm
	id := time.Now().Format("200601021504")
	e := &execution{
		id:         id,
		jobChan:    make(chan Job, c.podPoolSize),
		controller: c,
	}
	c.executions[id] = e

	fj := float64(jobs)
	c.progressStep = 100 / (fj * 3)
	atomic.StoreUint64(&c.progress, 0)
	c.prom.Pods(fj)

	for w := 1; w <= c.podPoolSize; w++ {
		go e.worker(w)
	}

	reportDir := filepath.Join(c.reportDir, id)

	if _, err := os.Stat(reportDir); os.IsNotExist(err) {
		err := os.MkdirAll(reportDir, 0o755)
		if err != nil {
			c.log.WithValues("dir", reportDir).Error(err, "error creating directory")
		}
	}

	if runtime.GOOS != "windows" {
		symlink := filepath.Join(c.reportDir, "latest")
		if _, err := os.Lstat(symlink); err == nil {
			err := os.Remove(symlink)
			if err != nil {
				c.log.WithValues("dir", symlink).Error(err, "error deleting latest link")
			}
		}
		err := os.Symlink(reportDir, symlink)
		if err != nil {
			c.log.WithValues("dir", symlink).Error(err, "error creating latest link")
		}
	}
	f, _ := strconv.ParseFloat(id, 64)

	c.prom.ExecutionStarted(f)
	return id
}

// AllAdded start the processing
func (c *controller) AllAdded(executionID string) error {
	e, err := c.forID(executionID)
	if err != nil {
		return err
	}

	files, err := ioutil.ReadDir(c.reportDir)
	if err != nil {
		c.log.WithValues("dir ", c.reportDir).Error(err, "could not list report dir files")
		return err
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime().Before(files[j].ModTime())
	})

	if len(files) > c.reportHistory {
		pruneCnt := len(files) - c.reportHistory
		for i := 0; i < pruneCnt; i++ {
			name := files[i].Name()
			// delete the execution
			delete(c.executions, name)
			c.prom.Prune(name)

			dir := filepath.Join(c.reportDir, name)
			c.log.WithValues("dir", dir).Info("deleting report directory")
			err = os.RemoveAll(dir)
			if err != nil {
				c.log.WithValues("dir", dir).Error(err, "could delete report directory")
			}
		}
	}
	close(e.jobChan)
	return nil
}

func (e *execution) worker(id int) {
	l := log.WithName("worker").WithValues("workerID", id)
	l.V(4).Info("initialized")
	for job := range e.jobChan {
		l.V(4).Info("process job", "jobID", job.ID(), "nodeName", job.Node())
		job.CreatePod()

		p, err := e.pod(job.Node())
		if err != nil {
			return
		}
		p.started = time.Now()
		p.status = "Started"
		e.controller.addProgress(1)

		for p.terminated == nil {
			time.Sleep(time.Second)
		}
		e.controller.addProgress(1)
		l.WithValues("jobID", job.ID(), "nodeName", job.Node(), "progress", e.controller.getProgress()).Info("job terminated")
	}
}

// AddPod add a new pod
func (c *controller) AddPod(job Job) error {
	e, err := c.forID(job.ID())
	if err != nil {
		return err
	}
	c.nodes[job.Node()] = true
	e.Store(job.Node(), &pod{
		node: job.Node(),
	})
	e.jobChan <- job
	return nil
}

// PodTerminated pod was terminated
func (c *controller) PodTerminated(executionID, node string, phase corev1.PodPhase) error {
	p, err := c.podForID(executionID, node)
	if err != nil {
		return err
	}
	if p.terminated != nil {
		return nil
	}
	c.addProgress(1)
	t := time.Now()
	p.terminated = &t
	p.status = string(phase)
	c.prom.Duration(node, executionID, float64(t.Sub(p.started).Milliseconds()))

	l := c.log.WithValues(
		"result ", phase,
		"node", node,
		"reports", p.reportReceived != nil,
		"progress", c.getProgress(),
	)

	// if not successful or not report received report an error
	if phase != corev1.PodSucceeded || p.reportReceived == nil {

		msg := "pod was not successful"
		if p.reportReceived == nil {
			msg = "did not receive report"
		}
		c.prom.ProcessingFinished(node, executionID, true)
		l.Info(msg)
	} else {
		l.Info("pod successful")
	}

	return nil
}

// ReportReceived report was received
func (c *controller) ReportReceived(executionID, node string, processingError error, results metrics.Results) {
	for k := range results {
		for _, r := range results[k] {
			c.prom.MetricFor(executionID, node, k, r)
		}
	}
	c.prom.ProcessingFinished(node, executionID, processingError != nil)

	e, err := c.forID(executionID)
	if err != nil {
		return
	}

	p, err := e.pod(node)
	if err != nil {
		return
	}

	t := time.Now()
	p.reportReceived = &t
	p.status = "ReportReceived"
}

func (c *controller) Has(node string, executionID string) bool {
	if _, ok := c.nodes[node]; !ok {
		return false
	}
	_, ok := c.executions[executionID]
	return ok
}

func (c *controller) forID(id string) (*execution, error) {
	e, ok := c.executions[id]
	if !ok {
		return nil, &ExecutionIDNotFound{Err: fmt.Errorf("execution with id: %q not found", id)}
	}
	return e, nil
}

func (c *controller) podForID(id, node string) (*pod, error) {
	e, err := c.forID(id)
	if err != nil {
		return nil, err
	}

	return e.pod(node)
}

type execution struct {
	sync.Map
	id         string
	jobChan    chan Job
	controller *controller
}

func (e *execution) pod(node string) (*pod, error) {
	p, ok := e.Load(node)
	if !ok {
		return nil, &ExecutionIDNotFound{Err: fmt.Errorf("pod for node: %q is not registered", node)}
	}
	return p.(*pod), nil
}

type pod struct {
	node           string
	started        time.Time
	terminated     *time.Time
	reportReceived *time.Time
	status         string
}

// Job interface
type Job interface {
	CreatePod()
	ID() string
	Node() string
}
