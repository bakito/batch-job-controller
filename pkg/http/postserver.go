package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"

	"github.com/bakito/batch-job-controller/pkg/config"
	"github.com/bakito/batch-job-controller/pkg/lifecycle"
	"github.com/bakito/batch-job-controller/pkg/metrics"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	// CallbackBasePath callback path
	CallbackBasePath = "/report/:node/:executionID"
	// CallbackBaseResultSubPath result sub path
	CallbackBaseResultSubPath = "/result"
	// CallbackBaseFileSubPath file sub path
	CallbackBaseFileSubPath = "/file"
	// CallbackBaseEventSubPath event sub path
	CallbackBaseEventSubPath = "/event"

	// FileName query parameter name
	FileName = "name"
)

var log = ctrl.Log.WithName("http-server")

// GenericAPIServer prepare the generic api server
func GenericAPIServer(port int, cfg *config.Config) manager.Runnable {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	s := &PostServer{
		Server: Server{
			Port:    port,
			Kind:    "internal",
			Handler: r,
		},
		ReportPath: cfg.ReportDirectory,
		DevMode:    cfg.DevMode,
	}

	rep := r.Group(CallbackBasePath)
	rep.Use(gin.Recovery(), s.middleware)

	rep.POST(CallbackBaseResultSubPath, s.postResult)

	rep.POST(CallbackBaseFileSubPath, s.postFile)

	rep.POST(CallbackBaseEventSubPath, s.postEvent)

	log.Info("starting callback",
		"port", port,
		"method", "POST",
		"path", fmt.Sprintf("%s%s", CallbackBasePath, CallbackBaseResultSubPath),
	)

	SetupProfiling(r)

	return s
}

// SetupProfiling setup profiling
func SetupProfiling(r *gin.Engine) {
	r.GET("/debug/pprof/", gin.WrapF(pprof.Index))
	r.GET("/debug/pprof/cmdline", gin.WrapF(pprof.Cmdline))
	r.GET("/debug/pprof/profile", gin.WrapF(pprof.Profile))
	r.GET("/debug/pprof/symbol", gin.WrapF(pprof.Symbol))
	r.GET("/debug/pprof/goroutine", gin.WrapH(pprof.Handler("goroutine")))
	r.GET("/debug/pprof/heap", gin.WrapH(pprof.Handler("heap")))
	r.GET("/debug/pprof/threadcreate", gin.WrapH(pprof.Handler("threadcreate")))
	r.GET("/debug/pprof/block", gin.WrapH(pprof.Handler("block")))
	r.GET("/debug/pprof/mutex", gin.WrapH(pprof.Handler("mutex")))
	r.GET("/debug/pprof/allocs", gin.WrapH(pprof.Handler("allocs")))
	r.GET("/debug/pprof/trace", gin.WrapH(pprof.Handler("trace")))
}

// PostServer post server
type PostServer struct {
	Server
	Controller    lifecycle.Controller
	ReportPath    string
	DevMode       bool
	EventRecorder record.EventRecorder
	Config        *config.Config
	Client        client.Reader
}

// InjectEventRecorder inject the event recorder
func (s *PostServer) InjectEventRecorder(er record.EventRecorder) {
	s.EventRecorder = er
}

// InjectController inject the controller
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

func (s *PostServer) postResult(ctx *gin.Context) {
	node, executionID := s.nodeAndID(ctx)
	postLog := log.WithValues(
		"node", node,
		"id", executionID,
	)
	body, err := ctx.GetRawData()
	if err != nil {
		ctx.String(http.StatusBadRequest, err.Error())
		postLog.Error(err, "error reading body")
		return
	}
	postLog = log.WithValues(
		"length", len(body),
	)

	results := new(metrics.Results)

	err = json.NewDecoder(bytes.NewReader(body)).Decode(&results)
	if err != nil {
		ctx.String(http.StatusBadRequest, err.Error())
		postLog.Error(err, "error decoding results json")
		return
	}

	err = results.Validate(s.Config)
	if err != nil {
		ctx.String(http.StatusBadRequest, err.Error())
		postLog.Error(err, "results is invalid")
		return
	}

	fileName, err := s.SaveFile(executionID, fmt.Sprintf("%s.json", node), body)
	postLog = postLog.WithValues(
		"name", filepath.Base(fileName),
		"path", fileName,
	)
	if err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
		postLog.Error(err, "error receiving file")
		return
	}
	s.Controller.ReportReceived(executionID, node, err, *results)
	postLog.Info("received report")
}

func (s *PostServer) postFile(ctx *gin.Context) {
	node, executionID := s.nodeAndID(ctx)
	postLog := log.WithValues(
		"node", node,
		"id", executionID,
	)

	form, _ := ctx.MultipartForm()
	if form != nil {
		cnt := 0
		for _, files := range form.File {
			for _, file := range files {

				// Upload the file to specific dst.
				if err := s.mkdir(executionID); err != nil {
					ctx.String(http.StatusInternalServerError, err.Error())
					postLog.Error(err, "error creating upload directory")
					return
				}

				err := ctx.SaveUploadedFile(file, filepath.Join(s.ReportPath, executionID, fmt.Sprintf("%s-%s", node, file.Filename)))
				if err != nil {
					ctx.String(http.StatusInternalServerError, err.Error())
					postLog.Error(err, "error saving file")
					return
				}
				cnt++
			}
		}
		postLog.Info(fmt.Sprintf("received %d file(s)", cnt))
	} else {
		body, err := ctx.GetRawData()
		if err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			postLog.Error(err, "error reading body")
			return
		}

		fileName := ctx.Query(FileName)
		if fileName == "" {
			_, params, _ := mime.ParseMediaType(ctx.GetHeader("Content-Disposition"))
			fileName = params["filename"]
		}
		if fileName == "" {
			fileName = uuid.New().String()

			fileName += s.evaluateExtension(ctx.Request)
		}

		fileName, err = s.SaveFile(executionID, fmt.Sprintf("%s-%s", node, fileName), body)
		postLog = log.WithValues(
			"name", filepath.Base(fileName),
			"path", fileName,
			"length", len(body),
		)
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			postLog.Error(err, "error receiving file")
			return
		}
		postLog.Info("received 1 file")
	}
}

func (s *PostServer) postEvent(ctx *gin.Context) {
	node, executionID := s.nodeAndID(ctx)
	postLog := log.WithValues(
		"node", node,
		"id", executionID,
	)
	body, err := ctx.GetRawData()
	if err != nil {
		ctx.String(http.StatusBadRequest, err.Error())
		postLog.Error(err, "error reading body")
		return
	}
	postLog = log.WithValues(
		"length", len(body),
	)

	event := new(Event)
	err = json.NewDecoder(bytes.NewReader(body)).Decode(&event)
	if err != nil {
		ctx.String(http.StatusBadRequest, fmt.Sprintf("error decoding event: %s", err.Error()))
		postLog.WithValues("result", string(body)).Error(err, "error decoding event")
		return
	}

	err = event.Validate()
	if err != nil {
		ctx.String(http.StatusBadRequest, err.Error())
		postLog.Error(err, "event is invalid")
		return
	}
	podName := s.Config.PodName(node, executionID)

	pod := &corev1.Pod{}
	err = s.Client.Get(ctx, client.ObjectKey{Namespace: s.Config.Namespace, Name: podName}, pod)

	if err != nil {
		err = fmt.Errorf("error finding pod: %w", err)
		ctx.String(http.StatusNotFound, err.Error())
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

func (s *PostServer) nodeAndID(ctx *gin.Context) (string, string) {
	node := ctx.Param("node")
	executionID := ctx.Param("executionID")
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
	if err := s.mkdir(executionID); err != nil {
		return "", err
	}
	fileName := filepath.Join(s.ReportPath, executionID, name)
	return fileName, ioutil.WriteFile(fileName, data, 0o600)
}

func (s *PostServer) mkdir(executionID string) error {
	return os.MkdirAll(filepath.Join(s.ReportPath, executionID), 0o755)
}
