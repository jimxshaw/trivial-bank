package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGinAdapter(t *testing.T) {
	// Define a mock that calls the next handler.
	m := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
			w.Write([]byte(" Middleware Triggered"))
		})
	}

	// Adapt the mock using GinAdapter.
	adapted := GinAdapter(m)

	// Simulate a request.
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	// This is the next context function to be executed.
	ctx.Next()

	adapted(ctx)

	require.Equal(t, " Middleware Triggered", w.Body.String())
}

func TestChain(t *testing.T) {
	mw1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("MW1 "))
			next.ServeHTTP(w, r)
		})
	}

	mw2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("MW2 "))
			next.ServeHTTP(w, r)
		})
	}

	final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Final"))
	})

	chained := Chain(mw1, mw2)(final)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	chained.ServeHTTP(w, req)

	require.Equal(t, "MW1 MW2 Final", w.Body.String())
}
