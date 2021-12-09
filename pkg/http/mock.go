package http

import (
	"fmt"
	"net/http/httputil"

	"github.com/bakito/batch-job-controller/pkg/config"
	"github.com/bakito/batch-job-controller/pkg/metrics"
	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// MockAPIServer prepare the mock api server
func MockAPIServer(port int) manager.Runnable {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	s := &mockServer{
		Server: Server{
			Port:    port,
			Kind:    "internal",
			Handler: r,
		},
	}

	rep := r.Group(CallbackBasePath)
	rep.POST(CallbackBaseResultSubPath, s.postResult)
	rep.POST(CallbackBaseFileSubPath, s.postFile)
	rep.POST(CallbackBaseEventSubPath, s.postEvent)

	log.Info("starting callback",
		"port", port,
		"method", "POST",
		"result", fmt.Sprintf("%s%s", CallbackBasePath, CallbackBaseResultSubPath),
		"file", fmt.Sprintf("%s%s", CallbackBasePath, CallbackBaseFileSubPath),
		"event", fmt.Sprintf("%s%s", CallbackBasePath, CallbackBaseEventSubPath),
	)

	return s
}

type mockServer struct {
	Server
}

func (s Server) postResult(ctx *gin.Context) {
	processPostResult(
		ctx,
		&config.Config{Name: "mock"},
		func(
			ctx *gin.Context,
			postLog logr.Logger,
			results *metrics.Results,
			node string,
			executionID string,
			body []byte,
		) error {
			return nil
		},
	)
}

func (s Server) postFile(ctx *gin.Context) {
	node, executionID := nodeAndID(ctx)
	postLog := log.WithValues(
		"node", node,
		"id", executionID,
	)
	postLog.Info("Got file(s)")
	b, _ := httputil.DumpRequest(ctx.Request, true)
	println(string(b))
}

func (s Server) postEvent(ctx *gin.Context) {
	processPostedEvent(
		ctx,
		&config.Config{Name: "mock"},
		func(ctx *gin.Context,
			postLog logr.Logger,
			podName string,
			event *Event,
		) error {
			return nil
		},
	)
}
