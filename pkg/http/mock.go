package http

import (
	"fmt"
	"mime/multipart"

	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/bakito/batch-job-controller/pkg/config"
	"github.com/bakito/batch-job-controller/pkg/metrics"
)

// MockAPIServer prepare the mock api server.
func MockAPIServer(port int) manager.Runnable {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	s := &mockServer{
		Server: &Server{
			Port:    port,
			Kind:    "internal",
			Handler: r,
			Log:     ctrl.Log.WithName("mock-server"),
			Config:  &config.Config{Name: "mock"},
		},
	}

	rep := r.Group(CallbackBasePath)
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

	return s
}

type mockServer struct {
	*Server
}

func (s *mockServer) postResult(ctx *gin.Context) {
	processPostResult(ctx, s.Server,
		func(*gin.Context, logr.Logger, *metrics.Results, string, string, []byte,
		) error {
			return nil
		},
	)
}

func (s *mockServer) postFile(ctx *gin.Context) {
	processPostedFiles(ctx, s.Server,
		func(*gin.Context, logr.Logger, string, string, *multipart.FileHeader) error {
			return nil
		},
		func(*gin.Context, logr.Logger, string, string, string, []byte) error {
			return nil
		},
	)
}

func (s *mockServer) postEvent(ctx *gin.Context) {
	processPostedEvent(ctx, s.Server,
		func(*gin.Context, logr.Logger, string, *Event) error {
			return nil
		},
	)
}
