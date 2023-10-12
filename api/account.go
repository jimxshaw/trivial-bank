package api

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	tl "github.com/jimxshaw/tracerlogger"
	auth "github.com/jimxshaw/trivial-bank/authentication/middleware"
	"github.com/jimxshaw/trivial-bank/authentication/token"
	db "github.com/jimxshaw/trivial-bank/db/sqlc"
	"github.com/lib/pq"
)

type listAccountsRequest struct {
	// https://gin-gonic.com/docs/examples/only-bind-query-string/
	PageID   int32 `form:"page_id" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=5,max=10"`
}

type getAccountRequest struct {
	// https://gin-gonic.com/docs/examples/bind-uri/
	ID int64 `uri:"id" binding:"required,min=1"`
}

type createAccountRequest struct {
	// https://pkg.go.dev/github.com/go-playground/validator/v10
	// Custom validation called currency registered in server.go.
	Currency string `json:"currency" binding:"required,currency"`
}

// Should NOT update the balance or currency here.
type updateAccountRequest struct {
	UserID int64 `json:"user_id" binding:"required"`
}

type deleteAccountRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

func (s *Server) listAccounts(ctx *gin.Context) {
	var req listAccountsRequest

	if err := ctx.ShouldBindQuery(&req); err != nil {
		errorResponse(tl.CodeBadRequest, ctx.Writer)
		return
	}

	authPayload := ctx.MustGet(string(auth.AuthPayloadKey)).(*token.Payload)

	params := db.ListAccountsParams{
		// Authorization Rule: users may only retrieve their own list of accounts.
		UserID: authPayload.UserID,
		Limit:  req.PageSize,
		Offset: (req.PageID - 1) * req.PageSize,
	}

	accounts, err := s.store.ListAccounts(ctx, params)
	if err != nil {
		errorResponse(tl.CodeInternalServerError, ctx.Writer)
		return
	}

	tl.RespondWithJSON(ctx.Writer, http.StatusOK, accounts)
}

func (s *Server) getAccount(ctx *gin.Context) {
	var req getAccountRequest

	if err := ctx.ShouldBindUri(&req); err != nil {
		errorResponse(tl.CodeBadRequest, ctx.Writer)
		return
	}

	account, err := s.store.GetAccount(ctx, req.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			errorResponse(tl.CodeNotFound, ctx.Writer)
			return
		}

		errorResponse(tl.CodeInternalServerError, ctx.Writer)
		return
	}

	authPayload := ctx.MustGet(string(auth.AuthPayloadKey)).(*token.Payload)

	// Authorization Rule: users may only retrieve their own account.
	if account.UserID != authPayload.UserID {
		err := errors.New("account does not belong to the authenticated user")
		tl.RespondWithError(ctx.Writer, http.StatusUnauthorized, err)
		return
	}

	tl.RespondWithJSON(ctx.Writer, http.StatusOK, account)
}

func (s *Server) createAccount(ctx *gin.Context) {
	var req createAccountRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		errorResponse(tl.CodeBadRequest, ctx.Writer)
		return
	}

	// Retrieve the "payload" stored in the context by a certain key.
	// Then use type assertion to assert that the value is
	// a pointer to our defined Payload struct.
	authPayload := ctx.MustGet(string(auth.AuthPayloadKey)).(*token.Payload)

	params := db.CreateAccountParams{
		// Authorization Rule: users may only create accounts for themselves.
		UserID:   authPayload.UserID,
		Currency: req.Currency,
		Balance:  0,
	}

	account, err := s.store.CreateAccount(ctx, params)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "foreign_key_violation", "unique_violation":
				errorResponse(tl.CodeForbidden, ctx.Writer)
				return
			}
		}
		errorResponse(tl.CodeInternalServerError, ctx.Writer)
		return
	}

	tl.RespondWithJSON(ctx.Writer, http.StatusOK, account)
}

func (s *Server) updateAccount(ctx *gin.Context) {
	var req updateAccountRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		errorResponse(tl.CodeBadRequest, ctx.Writer)
		return
	}

	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		errRes := tl.ResponseError{}
		errRes.AddValidationError(
			tl.CodeRouteVariableRequired,
			"id",
			"id route param missing or not a number",
		)
		tl.RespondWithError(ctx.Writer, http.StatusBadRequest, errRes)
		return
	}

	params := db.UpdateAccountParams{
		ID:     id,
		UserID: req.UserID,
	}

	account, err := s.store.UpdateAccount(ctx, params)
	if err != nil {
		errorResponse(tl.CodeInternalServerError, ctx.Writer)
		return
	}

	tl.RespondWithJSON(ctx.Writer, http.StatusOK, account)
}

func (s *Server) deleteAccount(ctx *gin.Context) {
	var req deleteAccountRequest

	if err := ctx.ShouldBindUri(&req); err != nil {
		errorResponse(tl.CodeBadRequest, ctx.Writer)
		return
	}

	err := s.store.DeleteAccount(ctx, req.ID)
	if err != nil {
		errorResponse(tl.CodeInternalServerError, ctx.Writer)
		return
	}

	tl.RespondWithJSON(ctx.Writer, http.StatusOK, map[string]string{"message": "account deleted"})
}
