package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	tl "github.com/jimxshaw/tracerlogger"
	"github.com/jimxshaw/tracerlogger/tracer"
	db "github.com/jimxshaw/trivial-bank/db/sqlc"
	mw "github.com/jimxshaw/trivial-bank/util/middleware"
)

// Server serves HTTP requests for our application.
type Server struct {
	store  db.Store
	router *gin.Engine
}

func NewServer(store db.Store) *Server {
	s := &Server{store: store}
	r := gin.Default()

	/* Middlewares */
	r.Use(mw.GinAdapter(tracer.TraceMiddleware()))

	/* Validators */
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("currency", validCurrency)
	}

	/* Routes */
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

	// Users
	r.POST("/users", s.createUser)

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

// errorResponse has common response for handler errors.
func errorResponse(err error, w http.ResponseWriter) {
	errRes, ok := err.(tl.Error)
	if !ok {
		tl.CodeInternalServerError.
			Respond(w, http.StatusInternalServerError, nil)
		return
	}

	switch errRes.CodeError() {
	case tl.CodeInternalServerError:
		errRes.Respond(w, http.StatusInternalServerError, nil)
	case tl.CodeBadRequest:
		errRes.Respond(w, http.StatusBadRequest, nil)
	case tl.CodeForbidden:
		errRes.Respond(w, http.StatusForbidden, nil)
	case tl.CodeNotFound:
		errRes.Respond(w, http.StatusNotFound, nil)
	}
}
