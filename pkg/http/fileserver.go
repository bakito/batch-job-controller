package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/bakito/batch-job-controller/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// StaticFileServer prepare the static file server
func StaticFileServer(port int, cfg *config.Config) manager.Runnable {
	return &Server{
		Port:    port,
		Kind:    "public",
		Handler: http.FileServer(http.Dir(cfg.ReportDirectory)),
	}
}

// Server default server
type Server struct {
	Port    int
	Kind    string
	Handler http.Handler
}

// Start the server
func (s *Server) Start(ctx context.Context) error {
	log.Info("starting http server", "port", s.Port, "type", s.Kind)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%v", s.Port),
		Handler: s.Handler,
	}

	idleConnsClosed := make(chan struct{})
	go func() {
		<-ctx.Done()
		log.Info("shutting down server")

		if err := srv.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout
			log.Error(err, "error shutting down the HTTP server")
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
