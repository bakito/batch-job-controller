package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	errorMiddlewareNotAcceptable = "node / execution ID not allowed"
)

func (s *PostServer) middleware(ctx *gin.Context) {
	if s.Controller != nil {
		if !s.Controller.Has(nodeAndID(ctx)) && !s.Config.DevMode {
			ctx.String(http.StatusNotAcceptable, errorMiddlewareNotAcceptable)
			ctx.Abort()
			return
		}
	}
	ctx.Next()
}
