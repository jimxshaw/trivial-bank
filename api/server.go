package api

import (
	"github.com/gin-gonic/gin"
	db "github.com/jimxshaw/trivial-bank/db/sqlc"
)

// Server serves HTTP requests for our application.
type Server struct {
	store  *db.Store
	router *gin.Engine
}

func NewServer(store *db.Store) *Server {
	s := &Server{store: store}
	r := gin.Default()

	r.POST("/accounts", s.createAccount)

	s.router = r

	return s
}

func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}
