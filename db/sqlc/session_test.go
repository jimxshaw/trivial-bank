package db

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jimxshaw/trivial-bank/util"
	"github.com/stretchr/testify/require"
)

func TestCreateSession(t *testing.T) {
	createRandomSession(t)
}

func createRandomSession(t *testing.T) Session {
	params := CreateSessionParams{
		ID:           uuid.New(),
		UserID:       util.RandomInt(1, 100_000),
		RefreshToken: util.RandomString(30),
		UserAgent:    util.RandomString(10),
		ClientIp:     util.RandomString(10),
		IsBlocked:    false,
		ExpiresAt:    time.Date(2050, time.January, 31, 12, 00, 00, 00, time.UTC),
	}

	query := `
		INSERT INTO sessions (
		id,
		user_id,
		refresh_token,
		user_agent,
		client_ip,
		is_blocked,
		expires_at
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7
	) RETURNING id, user_id, refresh_token, user_agent, client_ip, is_blocked, expires_at, created_at
	`

	rows := sqlmock.NewRows([]string{"id", "user_id", "refresh_token", "user_agent", "client_ip", "is_blocked", "expires_at", "created_at"}).
		AddRow(params.ID, params.UserID, params.RefreshToken, params.UserAgent, params.ClientIp, params.IsBlocked, params.ExpiresAt, time.Now())

	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(params.ID, params.UserID, params.RefreshToken, params.UserAgent, params.ClientIp, params.IsBlocked, params.ExpiresAt).
		WillReturnRows(rows)

	session, err := testQueries.CreateSession(context.Background(), params)
	require.NoError(t, err)
	require.NotEmpty(t, session)

	require.Equal(t, params.UserID, session.UserID)
	require.Equal(t, params.RefreshToken, session.RefreshToken)
	require.Equal(t, params.UserAgent, session.UserAgent)
	require.Equal(t, params.ClientIp, session.ClientIp)
	require.Equal(t, params.IsBlocked, session.IsBlocked)
	require.Equal(t, params.ExpiresAt, session.ExpiresAt)

	require.NotZero(t, session.ID)
	require.NotZero(t, session.CreatedAt)

	err = mock.ExpectationsWereMet()
	require.NoError(t, err)

	return session
}
