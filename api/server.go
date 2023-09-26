package api

import (
	"github.com/gin-gonic/gin"
	db "github.com/jimxshaw/trivial-bank/db/sqlc"
)

// Server serves HTTP requests for our application.
type Server struct {
	store  db.Store
	router *gin.Engine
}

func NewServer(store db.Store) *Server {
	s := &Server{store: store}
	r := gin.Default()

	// Accounts
	r.GET("/accounts", s.listAccounts)
	r.GET("/accounts/:id", s.getAccount)
	r.POST("/accounts", s.createAccount)

	// Entries
	// TODO: Add Entries routes here.

	// Transfers
	// TODO: Add Transfers routes here.

	s.router = r

	return s
}

// Start runs the HTTP server on the input address.
func (s *Server) Start(address string) error {
	// TODO: Add graceful shutdown logic.
	return s.router.Run(address)
}

func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}