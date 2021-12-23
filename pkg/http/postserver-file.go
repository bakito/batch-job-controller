package http

import (
	"fmt"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (s *PostServer) postFile(ctx *gin.Context) {
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

			fileName += s.evaluateExtension(ctx.Request)
		}

		fileName, err = s.SaveFile(executionID, fmt.Sprintf("%s-%s", node, fileName), body)
		postLog = postLog.WithValues(
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
