package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GinAdapter adapts an http middleware to be compatible with the gin framework.
func GinAdapter(mw func(http.Handler) http.Handler) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx.Request = r
			ctx.Next()
		})).ServeHTTP(ctx.Writer, ctx.Request)
	}
}
