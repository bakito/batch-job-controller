package http

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/bakito/batch-job-controller/pkg/config"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	envHealthCheckTimeout = "SELF_HEALTH_CHECK_CONNECTION_TIMEOUT"
)

// StaticFileServer prepare the static file server
func StaticFileServer(port int, cfg *config.Config) manager.Runnable {
	return &Server{
		Port:    port,
		Kind:    "public",
		Handler: http.FileServer(http.Dir(cfg.ReportDirectory)),
		Log:     ctrl.Log.WithName("file-server"),
		Config:  cfg,
	}
}

// Server default server
type Server struct {
	Port    int
	Kind    string
	Handler http.Handler
	Log     logr.Logger
	Config  *config.Config
}

// Start the server
func (s *Server) Start(ctx context.Context) error {
	s.Log.Info("starting http server", "port", s.Port, "type", s.Kind)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%v", s.Port),
		Handler: s.Handler,
	}

	idleConnsClosed := make(chan struct{})
	go func() {
		<-ctx.Done()
		s.Log.Info("shutting down server")

		if err := srv.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout
			s.Log.Error(err, "error shutting down the HTTP server")
		}
		close(idleConnsClosed)
	}()

	err := srv.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	<-idleConnsClosed
	return nil
}

// Name the name of the server
func (s *Server) Name() string {
	return "file-server"
}

// ReadyzCheck check if server is running
func (s *Server) ReadyzCheck() healthz.Checker {
	return s.HealthzCheck()
}

// HealthzCheck check if server is running
func (s *Server) HealthzCheck() healthz.Checker {
	timeout := time.Millisecond * 200
	if to, ok := os.LookupEnv(envHealthCheckTimeout); ok {
		if d, err := time.ParseDuration(to); err != nil {
			timeout = d
		} else {
			s.Log.WithValues(envHealthCheckTimeout, to, "default", timeout).
				Error(err, "could not parse self health check connection timeout; using default")
		}
	}
	return func(req *http.Request) error {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", s.Port), timeout)
		if err != nil {
			return err
		}
		if conn != nil {
			defer func() { _ = conn.Close() }()
		}
		return nil
	}
}
