package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/bakito/batch-job-controller/pkg/metrics"
	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
)

func (s *PostServer) postResult(ctx *gin.Context) {
	processPostResult(ctx, s.Server, s.postResultCallback)
}

func (s *PostServer) postResultCallback(ctx *gin.Context,
	postLog logr.Logger,
	results *metrics.Results,
	node string,
	executionID string,
	body []byte,
) error {
	fileName, err := s.SaveFile(executionID, fmt.Sprintf("%s.json", node), body)
	postLog = postLog.WithValues(
		"name", filepath.Base(fileName),
		"path", fileName,
	)
	if err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
		postLog.Error(err, "error receiving file")
		return err
	}
	s.Controller.ReportReceived(executionID, node, err, *results)
	return nil
}

type processPostResultCallback func(
	ctx *gin.Context,
	postLog logr.Logger,
	results *metrics.Results,
	node string,
	executionID string,
	body []byte,
) error

func processPostResult(ctx *gin.Context, s *Server, callback processPostResultCallback) {
	node, executionID := nodeAndID(ctx)
	postLog := s.Log.WithValues(
		"node", node,
		"id", executionID,
	)
	body, err := ctx.GetRawData()
	if err != nil {
		ctx.String(http.StatusBadRequest, err.Error())
		postLog.Error(err, "error reading body")
		return
	}
	postLog = postLog.WithValues(
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

	if callback(ctx, postLog, results, node, executionID, body) != nil {
		return
	}

	postLog.Info("received results")
}
