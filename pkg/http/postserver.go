package http

import (
	"fmt"
	"net/http/pprof"
	"os"

	"github.com/bakito/batch-job-controller/pkg/config"
	"github.com/bakito/batch-job-controller/pkg/lifecycle"
	"github.com/gin-gonic/gin"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
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

// GenericAPIServer prepare the generic api server
func GenericAPIServer(port int, cfg *config.Config) manager.Runnable {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	s := &PostServer{
		Server: &Server{
			Port:    port,
			Kind:    "internal",
			Handler: r,
			Log:     ctrl.Log.WithName("api-server"),
			Config:  cfg,
		},
		Config: cfg,
	}

	rep := r.Group(CallbackBasePath)
	rep.Use(gin.Recovery(), s.middleware)
	rep.POST(CallbackBaseResultSubPath, s.postResult)
	rep.POST(CallbackBaseFileSubPath, s.postFile)
	rep.POST(CallbackBaseEventSubPath, s.postEvent)

	s.Log.Info("starting callback",
		"port", port,
		"method", "POST",
		"result", fmt.Sprintf("%s%s", CallbackBasePath, CallbackBaseResultSubPath),
		"file", fmt.Sprintf("%s%s", CallbackBasePath, CallbackBaseFileSubPath),
		"event", fmt.Sprintf("%s%s", CallbackBasePath, CallbackBaseEventSubPath),
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
	*Server
	Controller    lifecycle.Controller
	Config        *config.Config
	EventRecorder record.EventRecorder
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

func nodeAndID(ctx *gin.Context) (string, string) {
	node := ctx.Param("node")
	executionID := ctx.Param("executionID")
	return node, executionID
}

// SaveFile save a received file
func (s *PostServer) SaveFile(executionID, name string, data []byte) (string, error) {
	if err := s.Config.MkReportDir(executionID); err != nil {
		return "", err
	}
	fileName := s.Config.ReportFileName(executionID, name)
	return fileName, os.WriteFile(fileName, data, 0o600)
}

// Name the name of the server
func (s *PostServer) Name() string {
	return "api-server"
}

// ReadyzCheck check if server is running
func (s *PostServer) ReadyzCheck() healthz.Checker {
	return s.Server.ReadyzCheck()
}

// HealthzCheck check if server is running
func (s *PostServer) HealthzCheck() healthz.Checker {
	return s.Server.HealthzCheck()
}
