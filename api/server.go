package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
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

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("currency", validCurrency)
	}

	// Health check
	r.GET("/health", s.healthCheck)

	// Accounts
	r.GET("/accounts", s.listAccounts)
	r.GET("/accounts/:id", s.getAccount)
	r.POST("/accounts", s.createAccount)
	r.PUT("/accounts/:id", s.updateAccount)
	r.DELETE("/accounts/:id", s.deleteAccount)

	// Entries
	r.GET("/entries", s.listEntries)
	r.GET("/entries/:id", s.getEntry)

	// Transfers
	r.GET("/transfers", s.listTransfers)
	r.GET("/transfers/:id", s.getTransfer)
	r.POST("/transfers", s.createTransfer)

	s.router = r

	return s
}

// Start runs the HTTP server on the input address.
func (s *Server) Start(address string) error {
	// TODO: Add graceful shutdown logic.
	return s.router.Run(address)
}

func (s *Server) healthCheck(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"status": "UP"})
}

func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}
