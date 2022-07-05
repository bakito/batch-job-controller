package http

import (
	"fmt"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
	"github.com/google/uuid"
)

func (s *PostServer) postFile(ctx *gin.Context) {
	processPostedFiles(ctx, s.Server, s.saveFormFilesCallback, s.saveBodyFileCallback)
}

func (s *PostServer) saveFormFilesCallback(ctx *gin.Context, postLog logr.Logger, executionID string, node string, file *multipart.FileHeader) error {
	if err := s.Config.MkReportDir(executionID); err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
		postLog.Error(err, "error creating upload directory")
		return err
	}

	err := ctx.SaveUploadedFile(file, s.Config.ReportFileName(executionID, fmt.Sprintf("%s-%s", node, file.Filename)))
	if err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
		postLog.Error(err, "error saving file")
		return err
	}
	return nil
}

func (s *PostServer) saveBodyFileCallback(ctx *gin.Context, postLog logr.Logger, executionID string, node string, fileName string, body []byte) error {
	fileName, err := s.SaveFile(executionID, fmt.Sprintf("%s-%s", node, fileName), body)
	postLog = postLog.WithValues(
		"name", filepath.Base(fileName),
		"path", fileName,
		"length", len(body),
	)
	if err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
		postLog.Error(err, "error receiving file")
		return err
	}
	return nil
}

type (
	saveFormFiles func(ctx *gin.Context, postLog logr.Logger, executionID string, node string, file *multipart.FileHeader) error
	saveBodyFile  func(ctx *gin.Context, postLog logr.Logger, executionID string, node string, fileName string, body []byte) error
)

func processPostedFiles(ctx *gin.Context, s *Server, ffCallback saveFormFiles, bfCallback saveBodyFile) {
	node, executionID := nodeAndID(ctx)
	postLog := s.Log.WithValues(
		"node", node,
		"id", executionID,
	)

	form, _ := ctx.MultipartForm()
	if form != nil {
		var names []string
		for _, files := range form.File {
			for _, file := range files {

				if ffCallback(ctx, postLog, executionID, node, file) != nil {
					return
				}

				names = append(names, file.Filename)
			}
		}
		postLog.WithValues(
			"names", strings.Join(names, ","),
		).Info(fmt.Sprintf("received %d file(s)", len(names)))
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

			fileName += evaluateExtension(ctx.Request)
		}

		if bfCallback(ctx, postLog, executionID, node, fileName, body) != nil {
			return
		}

		postLog.Info("received 1 file")
	}
}

func evaluateExtension(r *http.Request) string {
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
