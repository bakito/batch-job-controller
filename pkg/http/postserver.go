package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"net/http/pprof"
	"path/filepath"

	"github.com/bakito/batch-job-controller/pkg/config"
	"github.com/bakito/batch-job-controller/pkg/lifecycle"
	"github.com/bakito/batch-job-controller/pkg/metrics"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	// CallbackBasePath callback path
	CallbackBasePath = "/report/{node}/{executionID}"
	// CallbackBaseResultSubPath result sub path
	CallbackBaseResultSubPath = "/result"
	// CallbackBaseFileSubPath file sub path
	CallbackBaseFileSubPath = "/file"
	// CallbackBaseEventSubPath event sub path
	CallbackBaseEventSubPath = "/event"

	// FileName query parameter name
	FileName = "name"
)

var (
	log = ctrl.Log.WithName("http-server")
)

//GenericAPIServer prepare the generic api server
func GenericAPIServer(port int, reportPath string) manager.Runnable {

	r := mux.NewRouter()
	s := &PostServer{
		Server: Server{
			Port:    port,
			Kind:    "internal",
			Handler: r,
		},
		ReportPath: reportPath,
	}

	rep := r.PathPrefix(CallbackBasePath).Subrouter()
	rep.Use(s.middleware)

	rep.HandleFunc(CallbackBaseResultSubPath, s.postReport).
		Methods("POST").
		HeadersRegexp("Content-Type", "application/json")

	rep.HandleFunc(CallbackBaseFileSubPath, s.postFile).
		Methods("POST")

	rep.HandleFunc(CallbackBaseEventSubPath, s.postEvent).
		Methods("POST").
		HeadersRegexp("Content-Type", "application/json")

	log.Info("starting callback",
		"port", port,
		"method", "POST",
		"path", fmt.Sprintf("%s/%s", CallbackBasePath, CallbackBaseResultSubPath),
	)

	SetupProfiling(r)

	return s
}

// SetupProfiling setup profiling
func SetupProfiling(r *mux.Router) {
	r.HandleFunc("/debug/pprof/", pprof.Index)
	r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	r.HandleFunc("/debug/pprof/profile", pprof.Profile)
	r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	r.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	r.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	r.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	r.Handle("/debug/pprof/block", pprof.Handler("block"))
	r.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
	r.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))
	r.Handle("/debug/pprof/trace", pprof.Handler("trace"))
}

// PostServer post server
type PostServer struct {
	Server
	Controller    lifecycle.Controller
	ReportPath    string
	EventRecorder record.EventRecorder
	Config        *config.Config
	Client        client.Reader
}

// InjectEventRecorder inject the event recorder
func (s *PostServer) InjectEventRecorder(er record.EventRecorder) {
	s.EventRecorder = er
}

// InjectController inject the cache
func (s *PostServer) InjectController(c lifecycle.Controller) {
	s.Controller = c
}

// InjectReader inject the client reader
func (s *PostServer) InjectReader(reader client.Reader) {
	s.Client = reader
}

// InjectConfig inject the config
func (s *PostServer) InjectConfig(cfg *config.Config) {
	s.Config = cfg
}

func (s *PostServer) postReport(w http.ResponseWriter, r *http.Request) {

	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(r.Body)

	node, executionID := s.nodeAndID(r)

	postLog := log.WithValues(
		"node", node,
		"id", executionID,
		"length", len(buf.Bytes()),
	)

	results := new(metrics.Results)

	err := json.NewDecoder(bytes.NewReader(buf.Bytes())).Decode(&results)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		postLog.WithValues("result", string(buf.Bytes())).Error(err, "error decoding results json")
		return
	}

	err = results.Validate(s.Config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		postLog.Error(err, "results is invalid")
		return
	}

	fileName, err := s.SaveFile(executionID, fmt.Sprintf("%s.json", node), buf.Bytes())
	postLog = postLog.WithValues(
		"name", filepath.Base(fileName),
		"path", fileName,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		postLog.Error(err, "error receiving file")
		return
	}
	s.Controller.ReportReceived(executionID, node, err, *results)
	postLog.Info("received report")
}

func (s *PostServer) postFile(w http.ResponseWriter, r *http.Request) {
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(r.Body)

	fileName := r.URL.Query().Get(FileName)
	if fileName == "" {
		_, params, _ := mime.ParseMediaType(r.Header.Get("Content-Disposition"))
		fileName = params["filename"]
	}
	if fileName == "" {
		fileName = uuid.New().String()

		fileName += s.evaluateExtension(r)
	}
	node, executionID := s.nodeAndID(r)

	var err error
	fileName, err = s.SaveFile(executionID, fmt.Sprintf("%s-%s", node, fileName), buf.Bytes())
	postLog := log.WithValues(
		"node", node,
		"id", executionID,
		"name", filepath.Base(fileName),
		"path", fileName,
		"length", len(buf.Bytes()),
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		postLog.Error(err, "error receiving file")
		return
	}
	postLog.Info("received file")
}

func (s *PostServer) postEvent(w http.ResponseWriter, r *http.Request) {

	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(r.Body)

	node, executionID := s.nodeAndID(r)

	postLog := log.WithValues(
		"node", node,
		"id", executionID,
		"length", len(buf.Bytes()),
	)

	event := new(Event)
	err := json.NewDecoder(bytes.NewReader(buf.Bytes())).Decode(&event)
	if err != nil {
		http.Error(w, fmt.Sprintf("error decoding event: %s", err.Error()), http.StatusBadRequest)
		postLog.WithValues("result", string(buf.Bytes())).Error(err, "error decoding event")
		return
	}

	err = event.Validate()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		postLog.Error(err, "event is invalid")
		return
	}
	podName := s.Config.PodName(node, executionID)

	pod := &corev1.Pod{}
	err = s.Client.Get(r.Context(), client.ObjectKey{Namespace: s.Config.Namespace, Name: podName}, pod)

	if err != nil {
		err = fmt.Errorf("error finding pod: %v", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		postLog.Error(err, "")
		return
	}
	if len(event.Args) > 0 {
		s.EventRecorder.Eventf(pod, event.Type(), event.Reason, event.Message, event.args()...)
	} else {
		s.EventRecorder.Event(pod, event.Type(), event.Reason, event.Message)
	}
	postLog.Info("event created")
}

func (s *PostServer) nodeAndID(r *http.Request) (string, string) {
	vars := mux.Vars(r)
	node := vars["node"]
	executionID := vars["executionID"]
	return node, executionID
}

func (s *PostServer) evaluateExtension(r *http.Request) string {
	ct := r.Header.Get("Content-Type")

	mt, _, _ := mime.ParseMediaType(ct)
	if mt == "text/plain" {
		return ".txt"
	}
	ext, _ := mime.ExtensionsByType(ct)
	if len(ext) > 0 {
		return ext[0]
	}
	return ".file"
}

// SaveFile save a received file
func (s *PostServer) SaveFile(executionID, name string, data []byte) (string, error) {
	fileName := filepath.Join(s.ReportPath, executionID, name)
	return fileName, ioutil.WriteFile(fileName, data, 0644)
}
