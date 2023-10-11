package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jimxshaw/trivial-bank/authentication/token"
)

const (
	authHeaderKey  = "authorization"
	authTypeBearer = "bearer"
	authPayloadKey = "authorization_payload"
)

func AuthMiddleware(tokenGenerator token.Generator) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Header should be in <auth type> <token value> format:
		// E.g. Bearer v2.local.asdf
		authHeader := ctx.GetHeader(authHeaderKey)
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
		if authType != authTypeBearer {
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

		// Put payload in the context.
		ctx.Set(authPayloadKey, payload)

		// Forward the payload to the next handler.
		ctx.Next()
	}
}
