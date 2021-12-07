package client

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bakito/batch-job-controller/pkg/http"
	"github.com/bakito/batch-job-controller/pkg/job"
	"github.com/bakito/batch-job-controller/pkg/metrics"
	"gopkg.in/resty.v1"
)

type Client interface {
	SendResult(results *metrics.Results) error
	SendAsFile(name string, data []byte, contentType string) error
	SendFiles(filePaths ...string) error
	CreateEvent(isWaring bool, reason string, message string, args ...string) error
}

// Default get a default client with urls from env variables
func Default() Client {
	return New(
		os.Getenv(job.EnvCallbackServiceResultURL),
		os.Getenv(job.EnvCallbackServiceFileURL),
		os.Getenv(job.EnvCallbackServiceEventURL),
	)
}

// New create a new client
func New(resultURL string, fileURL string, eventURL string) Client {
	return &client{
		resultURL: resultURL,
		fileURL:   fileURL,
		eventURL:  eventURL,
		client:    resty.New().SetHeader("Content-Type", "application/json; charset=utf-8"),
	}
}

func (c client) SendResult(results *metrics.Results) error {
	return handleResponse(c.client.R().SetBody(results).SetContentLength(true).Post(c.resultURL))
}

func (c client) CreateEvent(isWaring bool, reason string, message string, args ...string) error {
	return handleResponse(c.client.R().SetBody(&http.Event{
		Waring:  isWaring,
		Reason:  reason,
		Message: message,
		Args:    args,
	}).SetContentLength(true).Post(c.eventURL))
}

func (c client) SendAsFile(name string, data []byte, contentType string) error {
	p := c.client.R().SetHeader("Content-Disposition", fmt.Sprintf(`attachment;filename="%s"`, name))
	if contentType != "" {
		p = p.SetHeader("Content-Type", contentType)
	}
	return handleResponse(p.SetBody(data).SetContentLength(true).Post(c.fileURL))
}

func handleResponse(resp *resty.Response, err error) error {
	if resp != nil && resp.StatusCode() != 200 {
		return &httpError{status: resp.Status(), message: resp.String()}
	}
	return err
}

func (c client) SendFiles(filePaths ...string) error {
	files := make(map[string]string)
	for _, path := range filePaths {
		files[filepath.Base(path)] = path
	}
	resp, err := c.client.R().SetFiles(files).Post(c.fileURL)
	if resp.StatusCode() != 200 {
		err = &httpError{status: resp.Status(), message: resp.String()}
	}
	return err
}

type client struct {
	resultURL string
	fileURL   string
	eventURL  string
	client    *resty.Client
}

type httpError struct {
	message string
	status  string
}

func (h httpError) Error() string {
	return fmt.Sprintf("%s: %s", h.status, h.message)
}
