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

// Chain combines middlewares into a single middleware.
func Chain(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(final http.Handler) http.Handler {
		// Reverse loop is used to ensure the order of middlewares as intended.
		// E.g. the first middleware in the list will be the first one that executes
		// on any incoming request.
		for i := len(middlewares) - 1; i >= 0; i-- {
			// Pass final as argument into each middleware function iteration.
			final = middlewares[i](final)
		}
		return final
	}
}
