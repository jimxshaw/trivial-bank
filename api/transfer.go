package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	tl "github.com/jimxshaw/tracerlogger"
	auth "github.com/jimxshaw/trivial-bank/authentication/middleware"
	"github.com/jimxshaw/trivial-bank/authentication/token"
	db "github.com/jimxshaw/trivial-bank/db/sqlc"
)

type listTransfersRequest struct {
	PageID   int32 `form:"page_id" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=5,max=10"`
}

type getTransferRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

type createTransferRequest struct {
	FromAccountID int64  `json:"from_account_id" binding:"required,min=1"`
	ToAccountID   int64  `json:"to_account_id" binding:"required,min=1"`
	Amount        int64  `json:"amount" binding:"required,gt=0"`
	Currency      string `json:"currency" binding:"required,currency"`
}

func (s *Server) listTransfers(ctx *gin.Context) {
	var req listTransfersRequest

	if err := ctx.ShouldBindQuery(&req); err != nil {
		errorResponse(tl.CodeBadRequest, ctx.Writer)
		return
	}

	params := db.ListTransfersParams{
		Limit:  req.PageSize,
		Offset: (req.PageID - 1) * req.PageSize,
	}

	transfers, err := s.store.ListTransfers(ctx, params)
	if err != nil {
		errorResponse(tl.CodeInternalServerError, ctx.Writer)
		return
	}

	tl.RespondWithJSON(ctx.Writer, http.StatusOK, transfers)
}

func (s *Server) getTransfer(ctx *gin.Context) {
	var req getTransferRequest

	if err := ctx.ShouldBindUri(&req); err != nil {
		errorResponse(tl.CodeBadRequest, ctx.Writer)
		return
	}

	transfer, err := s.store.GetTransfer(ctx, req.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			errorResponse(tl.CodeNotFound, ctx.Writer)
			return
		}

		errorResponse(tl.CodeInternalServerError, ctx.Writer)
		return
	}

	fromAccount, err := s.store.GetAccount(ctx, transfer.FromAccountID)
	if err != nil {
		errorResponse(tl.CodeInternalServerError, ctx.Writer)
		return
	}

	toAccount, err := s.store.GetAccount(ctx, transfer.ToAccountID)
	if err != nil {
		errorResponse(tl.CodeInternalServerError, ctx.Writer)
		return
	}

	authPayload := ctx.MustGet(string(auth.AuthPayloadKey)).(*token.Payload)

	// Authorization Rule: users may get a transfer only if their account
	// is involved as the sender or as the receiver of funds.
	if fromAccount.UserID != authPayload.UserID && toAccount.UserID != authPayload.UserID {
		errorResponse(tl.CodeUnauthorized, ctx.Writer)
		return
	}

	tl.RespondWithJSON(ctx.Writer, http.StatusOK, transfer)
}

func (s *Server) createTransfer(ctx *gin.Context) {
	var req createTransferRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		errorResponse(tl.CodeBadRequest, ctx.Writer)
		return
	}

	fromAccount, isValid := s.isValidAccount(ctx, req.FromAccountID, req.Currency)
	if !isValid {
		return
	}

	authPayload := ctx.MustGet(string(auth.AuthPayloadKey)).(*token.Payload)

	// Authorization Rule: users may only send money from their own accounts.
	if fromAccount.UserID != authPayload.UserID {
		err := errors.New("from account does not belong to the authenticated user")
		tl.RespondWithError(ctx.Writer, http.StatusUnauthorized, err)
		return
	}

	_, isValid = s.isValidAccount(ctx, req.ToAccountID, req.Currency)
	if !isValid {
		return
	}

	params := db.TransferTxParams{
		FromAccountID: req.FromAccountID,
		ToAccountID:   req.ToAccountID,
		Amount:        req.Amount,
	}

	result, err := s.store.TransferTx(ctx, params)
	if err != nil {
		errorResponse(tl.CodeInternalServerError, ctx.Writer)
		return
	}

	tl.RespondWithJSON(ctx.Writer, http.StatusOK, result)
}

func (s *Server) isValidAccount(ctx *gin.Context, accountID int64, currency string) (db.Account, bool) {
	account, err := s.store.GetAccount(ctx, accountID)
	if err != nil {
		if err == sql.ErrNoRows {
			errorResponse(tl.CodeNotFound, ctx.Writer)
			return account, false
		}

		errorResponse(tl.CodeInternalServerError, ctx.Writer)
		return account, false
	}

	if account.Currency != currency {
		err := fmt.Errorf("account [%d] currency mismatch: %s vs %s", account.ID, account.Currency, currency)
		errRes := tl.ResponseError{}
		errRes.Respond(ctx.Writer, http.StatusBadRequest, err)
		return account, false
	}

	return account, true
}
