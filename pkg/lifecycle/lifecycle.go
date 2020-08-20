package lifecycle

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/bakito/batch-job-controller/pkg/config"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	log = ctrl.Log.WithName("lifecycle")
)

//NewCache get a new cache
func NewCache(cfg *config.Config, prom *Collector) Cache {
	return &cache{
		executions:    make(map[string]*execution),
		prom:          prom,
		log:           log.WithName("cache"),
		reportHistory: cfg.ReportHistory + 1, // 1+ for latest
		reportDir:     cfg.ReportDirectory,
		podPoolSize:   cfg.PodPoolSize,
		config:        *cfg,
	}
}

//Cache interface
type Cache interface {
	NewExecution() string
	AllAdded(executionID string) error
	AddPod(job Job) error
	PodTerminated(executionID, node string, phase corev1.PodPhase) error
	ReportReceived(executionID, node string, processingError error, results map[string][]Result)
	Config() config.Config
}

type cache struct {
	prom          *Collector
	executions    map[string]*execution
	log           logr.Logger
	reportDir     string
	reportHistory int
	podPoolSize   int
	config        config.Config
}

// verify interface is implemented
var _ Cache = &cache{}

// Config get the config
func (c *cache) Config() config.Config {
	return c.config
}

// NewExecution setup a new execution
func (c *cache) NewExecution() string {
	//                             yyyyMMddHHmmss
	id := time.Now().Format("20060102150400")
	e := &execution{
		id:      id,
		jobChan: make(chan Job, c.podPoolSize),
	}
	c.executions[id] = e

	for w := 1; w <= c.podPoolSize; w++ {
		go e.worker(w)
	}

	reportDir := filepath.Join(c.reportDir, id)

	if _, err := os.Stat(reportDir); os.IsNotExist(err) {
		err := os.MkdirAll(reportDir, 0755)
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
	return id
}

//  AllAdded start the processing
func (c *cache) AllAdded(executionID string) error {
	e, err := c.forID(executionID)
	if err != nil {
		return err
	}

	cnt := e.length()
	c.prom.pods(cnt)

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
			dir := c.reportDir + "/" + files[i].Name()
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
		job.Process()

		p, err := e.pod(job.Node())
		if err != nil {
			return
		}
		p.started = time.Now()
		p.status = "Started"

		for p.terminated == nil {
			time.Sleep(time.Second)
		}
		l.V(4).Info("job terminated", "jobID", job.ID(), "nodeName", job.Node())
	}
}

// AddPod add a new pod
func (c *cache) AddPod(job Job) error {
	e, err := c.forID(job.ID())
	if err != nil {
		return err
	}
	e.Store(job.Node(), &pod{
		node: job.Node(),
	})
	e.jobChan <- job
	return nil
}

// PodTerminated pod was terminated
func (c *cache) PodTerminated(executionID, node string, phase corev1.PodPhase) error {
	p, err := c.podForID(executionID, node)
	if err != nil {
		return err
	}
	t := time.Now()
	p.terminated = &t
	p.status = string(phase)
	c.prom.duration(node, executionID, float64(t.Sub(p.started).Milliseconds()))

	// if not successful or not report received report an error
	if phase != corev1.PodSucceeded || p.reportReceived == nil {

		msg := "pod was not successful"
		if p.reportReceived == nil {
			msg = "did not receive report"
		}
		c.prom.processingError(node, executionID, true)
		c.log.WithValues("result ", phase, "node", node, "reports", p.reportReceived != nil).Info(msg)
	} else {
		c.log.WithValues("result ", phase, "node", node).Info("pod successful")
	}

	return nil
}

// ReportReceived report was received
func (c *cache) ReportReceived(executionID, node string, processingError error, results map[string][]Result) {
	for k := range results {
		for _, r := range results[k] {
			c.prom.metricFor(executionID, node, k, r)
		}
	}
	c.prom.processingError(node, executionID, processingError != nil)

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

func (c *cache) forID(id string) (*execution, error) {
	e, ok := c.executions[id]
	if !ok {
		return nil, ExecutionIDNotFound(fmt.Errorf("execution with id: '%s' not found", id))
	}
	return e, nil
}

func (c *cache) podForID(id, node string) (*pod, error) {
	e, err := c.forID(id)
	if err != nil {
		return nil, err
	}

	return e.pod(node)
}

type execution struct {
	sync.Map
	id      string
	jobChan chan Job
}

func (e *execution) length() float64 {
	var length float64 = 0

	e.Map.Range(func(_, _ interface{}) bool {
		length++

		return true
	})
	return length
}

func (e *execution) pod(node string) (*pod, error) {
	p, ok := e.Load(node)
	if !ok {
		return nil, fmt.Errorf("pod for node: '%s' is not registered", node)
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
	Process()
	ID() string
	Node() string
}
