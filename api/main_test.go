package api

import (
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMain(m *testing.M) {
	// In test mode, Gin will not print
	// logs in order to keep output clean.
	gin.SetMode(gin.TestMode)

	os.Exit(m.Run())
}
