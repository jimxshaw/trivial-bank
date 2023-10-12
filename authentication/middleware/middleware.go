package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/jimxshaw/trivial-bank/authentication/token"
)

type contextKey string

const (
	authHeaderKey  string     = "authorization"
	authTypeBearer string     = "bearer"
	AuthPayloadKey contextKey = "authorization_payload"
)

func AuthMiddleware(tokenGenerator token.Generator) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// The header should be in <auth type> <token value> format:
			// E.g. Bearer abcdefg
			authHeader := r.Header.Get(authHeaderKey)
			if authHeader == "" {
				http.Error(w, "authorization header is missing", http.StatusUnauthorized)
				return
			}

			fields := strings.Fields(authHeader)
			if len(fields) < 2 {
				http.Error(w, "invalid authorization header format", http.StatusUnauthorized)
				return
			}

			authType := strings.ToLower(fields[0])
			if authType != authTypeBearer {
				http.Error(w, fmt.Sprintf("unsupported authorization type %s", authType), http.StatusUnauthorized)
				return
			}

			accessToken := fields[1]

			payload, err := tokenGenerator.ValidateToken(accessToken)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			// Store the payload in the request's context
			ctx := context.WithValue(r.Context(), AuthPayloadKey, payload)
			r = r.WithContext(ctx)

			// Move to the next handler.
			next.ServeHTTP(w, r)
		})
	}
}
