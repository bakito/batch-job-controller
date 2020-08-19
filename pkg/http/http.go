package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"net/http/pprof"
	"path/filepath"

	"github.com/bakito/batch-job-controller/pkg/lifecycle"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	CallbackBasePath          = "/report/{node}/{executionID}"
	CallbackBaseResultSubPath = "/result"
	CallbackBaseFileSubPath   = "/file"

	FileName = "name"
)

var (
	log = ctrl.Log.WithName("http-server")
)

//StaticFileServer prepare the static file server
func StaticFileServer(port int, path string) manager.Runnable {
	return &Server{
		Port:    port,
		Kind:    "public",
		Handler: http.FileServer(http.Dir(path)),
	}
}

//GenericAPIServer prepare the generic api server
func GenericAPIServer(port int, reportPath string, cache lifecycle.Cache) manager.Runnable {

	r := mux.NewRouter()
	s := &PostServer{
		Server: Server{
			Port:    port,
			Kind:    "internal",
			Handler: r,
		},
		ReportPath: reportPath,
		Cache:      cache,
	}

	rep := r.PathPrefix(CallbackBasePath).Subrouter()

	rep.HandleFunc(CallbackBaseResultSubPath, s.postGenericReport).
		Methods("POST").
		HeadersRegexp("Content-Type", "application/json")

	rep.HandleFunc(CallbackBaseFileSubPath, s.postGenericFile).
		Methods("POST")

	log.Info("starting callback",
		"port", port,
		"method", "POST",
		"path", fmt.Sprintf("%s/%s", CallbackBasePath, CallbackBaseResultSubPath),
	)

	SetupProfiling(r)

	return s
}

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

type PostServer struct {
	Server
	Cache      lifecycle.Cache
	ReportPath string
}

type Server struct {
	Port    int
	Kind    string
	Handler http.Handler
}

// Start the server
func (s *Server) Start(stop <-chan struct{}) error {
	log.Info("starting http server", "port", s.Port, "type", s.Kind)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%v", s.Port),
		Handler: s.Handler,
	}

	idleConnsClosed := make(chan struct{})
	go func() {
		<-stop
		log.Info("shutting down server")

		if err := srv.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout
			log.Error(err, "error shutting down the HTTP server")
		}
		close(idleConnsClosed)
	}()

	err := srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}

	<-idleConnsClosed
	return nil
}

func (s *PostServer) postGenericReport(w http.ResponseWriter, r *http.Request) {

	result := make(map[string][]lifecycle.Result)

	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(r.Body)

	vars := mux.Vars(r)
	node := vars["node"]
	executionID := vars["executionID"]

	postLog := log.WithValues(
		"node", node,
		"id", executionID,
		"length", len(buf.Bytes()),
	)

	err := json.NewDecoder(bytes.NewReader(buf.Bytes())).Decode(&result)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		postLog.WithValues("result", string(buf.Bytes())).Error(err, "error decoding result json")
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
	s.Cache.ReportReceived(executionID, node, err, result)
	postLog.Info("received report")
}

func (s *PostServer) postGenericFile(w http.ResponseWriter, r *http.Request) {

	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(r.Body)

	vars := mux.Vars(r)
	fileName := r.URL.Query().Get(FileName)
	if fileName == "" {
		_, params, _ := mime.ParseMediaType(r.Header.Get("Content-Disposition"))
		fileName = params["filename"]
	}
	if fileName == "" {
		fileName = uuid.New().String()
	}

	node := vars["node"]
	executionID := vars["executionID"]

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
	postLog.WithValues("length", len(buf.Bytes())).Info("received file")
}

func (s *PostServer) SaveFile(executionID, name string, data []byte) (string, error) {
	fileName := filepath.Join(s.ReportPath, executionID, name)
	return fileName, ioutil.WriteFile(fileName, data, 0644)
}
