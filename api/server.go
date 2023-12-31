package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	tl "github.com/jimxshaw/tracerlogger"
	"github.com/jimxshaw/tracerlogger/tracer"
	auth "github.com/jimxshaw/trivial-bank/authentication/middleware"
	"github.com/jimxshaw/trivial-bank/authentication/token"
	db "github.com/jimxshaw/trivial-bank/db/sqlc"
	"github.com/jimxshaw/trivial-bank/util"
	mw "github.com/jimxshaw/trivial-bank/util/middleware"
)

// Server serves HTTP requests for our application.
type Server struct {
	store          db.Store
	config         util.Config
	tokenGenerator token.Generator
	router         *gin.Engine
}

func NewServer(store db.Store, config util.Config) (*Server, error) {
	tokenGenerator, err := token.NewPasetoGenerator(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token generator %w", err)
	}

	s := &Server{
		store:          store,
		config:         config,
		tokenGenerator: tokenGenerator,
	}

	/* Validators */
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("currency", validCurrency)
	}

	s.setupRouter()

	return s, nil
}

// Start runs the HTTP server on the input address.
func (s *Server) Start(address string) error {
	// TODO: Add graceful shutdown logic.
	return s.router.Run(address)
}

func (s *Server) healthCheck(ctx *gin.Context) {
	tl.RespondWithJSON(ctx.Writer, http.StatusOK, gin.H{"status": "UP"})
}

func (s *Server) setupRouter() {
	r := gin.Default()

	// Tracer will be applied to all routes.
	r.Use(mw.GinAdapter(tracer.TraceMiddleware()))

	/* Non-Authentication */
	// Users
	r.POST("/users", s.createUser)
	r.POST("/users/login", s.loginUser)
	r.POST("/tokens/renew_access", s.renewAccessToken)

	// Health check
	r.GET("/health", s.healthCheck)

	// TODO: Entries routes require authorization
	// roles to be implemented.
	r.GET("/entries", s.listEntries)
	r.GET("/entries/:id", s.getEntry)

	/* Authentication */
	authRoutes := r.Group("/")
	authRoutes.Use(auth.AuthGinMiddleware(s.tokenGenerator))

	// Accounts
	authRoutes.GET("/accounts", s.listAccounts)
	authRoutes.GET("/accounts/:id", s.getAccount)
	authRoutes.POST("/accounts", s.createAccount)
	authRoutes.PUT("/accounts/:id", s.updateAccount)
	authRoutes.DELETE("/accounts/:id", s.deleteAccount)

	// Transfers
	authRoutes.GET("/transfers", s.listTransfers)
	authRoutes.GET("/transfers/:id", s.getTransfer)
	authRoutes.POST("/transfers", s.createTransfer)

	s.router = r
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
	case tl.CodeUnauthorized:
		errRes.Respond(w, http.StatusUnauthorized, nil)
	}
}
