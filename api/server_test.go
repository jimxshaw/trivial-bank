package api

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStart(t *testing.T) {
	finish, m := newStoreMock(t)
	defer finish()

	server := NewServer(m)
	go server.Start(":8080")

	// Give the server some time to start
	time.Sleep(1 * time.Second)

	resp, err := http.Get("http://localhost:8080/health")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}
