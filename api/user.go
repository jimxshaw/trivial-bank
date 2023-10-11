package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	tl "github.com/jimxshaw/tracerlogger"
	db "github.com/jimxshaw/trivial-bank/db/sqlc"
	"github.com/jimxshaw/trivial-bank/util"
	"github.com/lib/pq"
)

type createUserRequest struct {
	FirstName string `json:"first_name" binding:"required,min=2"`
	LastName  string `json:"last_name" binding:"required,min=2"`
	Email     string `json:"email" binding:"required,email"`
	Username  string `json:"username" binding:"required,alphanum,min=6"`
	Password  string `json:"password" binding:"required"`
}

type userResponse struct {
	FirstName         string    `json:"first_name"`
	LastName          string    `json:"last_name"`
	Email             string    `json:"email"`
	Username          string    `json:"username"`
	PasswordChangedAt time.Time `json:"password_changed_at"`
	CreatedAt         time.Time `json:"created_at"`
}

func newUserResponse(user db.User) userResponse {
	return userResponse{
		FirstName:         user.FirstName,
		LastName:          user.LastName,
		Email:             user.Email,
		Username:          user.Username,
		PasswordChangedAt: user.PasswordChangedAt,
		CreatedAt:         user.CreatedAt,
	}
}

func (s *Server) createUser(ctx *gin.Context) {
	var req createUserRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		tl.RespondWithError(ctx.Writer, http.StatusBadRequest, err)
		return
	}

	// Password must be validated.
	if !util.IsValidPassword(req.Password) {
		errRes := tl.ResponseError{}
		errRes.AddValidationError(
			tl.CodeFieldsValidation,
			"password",
			util.PasswordValidationMessage,
		)
		tl.RespondWithError(ctx.Writer, http.StatusBadRequest, errRes)
		return
	}

	// Password must be hashed.
	hash, err := util.HashPassword(req.Password)
	if err != nil {
		errorResponse(tl.CodeInternalServerError, ctx.Writer)
		return
	}

	params := db.CreateUserParams{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		Username:  req.Username,
		Password:  hash,
	}

	user, err := s.store.CreateUser(ctx, params)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				// Check if username or email already exists.
				tl.RespondWithError(ctx.Writer, http.StatusForbidden, pqErr)
				return
			}
		}
		errorResponse(tl.CodeInternalServerError, ctx.Writer)
		return
	}

	res := newUserResponse(user)

	tl.RespondWithJSON(ctx.Writer, http.StatusOK, res)
}

type loginUserRequest struct {
	Username string `json:"username" binding:"required,alphanum,min=6"`
	Password string `json:"password" binding:"required"`
}

type loginUserResponse struct {
	AccessToken string       `json:"access_token"`
	User        userResponse `json:"user"`
}
