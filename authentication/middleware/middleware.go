package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jimxshaw/trivial-bank/authentication/token"
)

type contextKey string

const (
	AuthHeaderKey  string     = "authorization"
	AuthPayloadKey contextKey = "authorization_payload"
	AuthTypeBearer string     = "bearer"
)

// AuthGinMiddleware creates a gin middleware for authorization.
func AuthGinMiddleware(tokenGenerator token.Generator) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader(AuthHeaderKey)

		if len(authHeader) == 0 {
			err := errors.New("authorization header is missing")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, err.Error())
			return
		}

		fields := strings.Fields(authHeader)
		if len(fields) < 2 {
			err := errors.New("invalid authorization header format")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, err.Error())
			return
		}

		authType := strings.ToLower(fields[0])
		if authType != AuthTypeBearer {
			err := fmt.Errorf("unsupported authorization type %s", authType)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, err.Error())
			return
		}

		accessToken := fields[1]
		payload, err := tokenGenerator.ValidateToken(accessToken)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, err.Error())
			return
		}

		ctx.Set(string(AuthPayloadKey), payload)
		ctx.Next()
	}
}
