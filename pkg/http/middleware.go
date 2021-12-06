package http

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

const (
	errorMiddlewareNotAcceptable = "node / execution ID not allowed"
)

func (s *PostServer) middleware(ctx *gin.Context) {
	if s.Controller != nil {
		if !s.Controller.Has(s.nodeAndID(ctx)) {
			ctx.String(http.StatusNotAcceptable, errorMiddlewareNotAcceptable)
			ctx.Abort()
			return
		}
	}
	ctx.Next()
}
