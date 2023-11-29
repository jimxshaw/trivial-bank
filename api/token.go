package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	tl "github.com/jimxshaw/tracerlogger"
)

type renewAccessTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type renewAccessTokenResponse struct {
	AccessToken          string    `json:"access_token"`
	AccessTokenExpiresAt time.Time `json:"access_token_expires_at"`
}

func (s *Server) renewAccessToken(ctx *gin.Context) {
	var req renewAccessTokenRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		errorResponse(tl.CodeBadRequest, ctx.Writer)
		return
	}

	refreshPayload, err := s.tokenGenerator.ValidateToken(req.RefreshToken)
	if err != nil {
		errorResponse(tl.CodeUnauthorized, ctx.Writer)
		return
	}

	session, err := s.store.GetSession(ctx, refreshPayload.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			errorResponse(tl.CodeNotFound, ctx.Writer)
			return
		}
		errorResponse(tl.CodeInternalServerError, ctx.Writer)
		return
	}

	if session.IsBlocked {
		ctx.JSON(http.StatusUnauthorized, fmt.Errorf("blocked session"))
		return
	}

	if session.UserID != refreshPayload.UserID {
		ctx.JSON(http.StatusUnauthorized, fmt.Errorf("incorrect session user"))
		return
	}

	if session.RefreshToken != req.RefreshToken {
		ctx.JSON(http.StatusUnauthorized, fmt.Errorf("mismatched session token"))
		return
	}

	if time.Now().After(session.ExpiresAt) {
		ctx.JSON(http.StatusUnauthorized, fmt.Errorf("expired session"))
		return
	}

	accessToken, accessPayload, err := s.tokenGenerator.GenerateToken(
		refreshPayload.UserID,
		s.config.AccessTokenDuration,
	)
	if err != nil {
		errorResponse(tl.CodeInternalServerError, ctx.Writer)
		return
	}

	res := renewAccessTokenResponse{
		AccessToken:          accessToken,
		AccessTokenExpiresAt: accessPayload.ExpiredAt,
	}

	tl.RespondWithJSON(ctx.Writer, http.StatusOK, res)
}
