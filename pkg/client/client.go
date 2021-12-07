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
		client:    resty.New().SetHeader("Content-Type", "application/json"),
	}
}

func (c client) SendResult(results *metrics.Results) error {
	_, err := c.client.R().SetBody(results).SetContentLength(true).Post(c.resultURL)
	return err
}

func (c client) CreateEvent(isWaring bool, reason string, message string, args ...string) error {
	_, err := c.client.R().SetBody(&http.Event{
		Waring:  isWaring,
		Reason:  reason,
		Message: message,
		Args:    args,
	}).SetContentLength(true).Post(c.eventURL)
	return err
}

func (c client) SendAsFile(name string, data []byte, contentType string) error {
	p := c.client.R().SetHeader("Content-Disposition", fmt.Sprintf(`attachment;filename="%s"`, name))
	if contentType != "" {
		p = p.SetHeader("Content-Type", contentType)
	}
	_, err := p.SetBody(data).SetContentLength(true).Post(c.fileURL)
	return err
}

func (c client) SendFiles(filePaths ...string) error {
	files := make(map[string]string)
	for _, path := range filePaths {
		files[filepath.Base(path)] = path
	}
	_, err := c.client.R().SetFiles(files).Post(c.fileURL)
	return err
}

type client struct {
	resultURL string
	fileURL   string
	eventURL  string
	client    *resty.Client
}
