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

func TestGetSession(t *testing.T) {
	session := createRandomSession(t)

	t.Run("get session by ID", func(t *testing.T) {
		query := `
			SELECT id, user_id, refresh_token, user_agent, client_ip, is_blocked, expires_at, created_at 
			FROM sessions
			WHERE id = $1 LIMIT
		`

		rows := sqlmock.NewRows([]string{"id", "user_id", "refresh_token", "user_agent", "client_ip", "is_blocked", "expires_at", "created_at"}).
			AddRow(session.ID, session.UserID, session.RefreshToken, session.UserAgent, session.ClientIp, session.IsBlocked, session.ExpiresAt, time.Now())

		mock.ExpectQuery(regexp.QuoteMeta(query)).
			WithArgs(session.ID).
			WillReturnRows(rows)

		sess, err := testQueries.GetSession(context.Background(), session.ID)
		require.NoError(t, err)
		require.NotEmpty(t, sess)

		requireGetSession(t, session, sess)
	})

}

func requireGetSession(t *testing.T, want Session, got Session) {
	require.Equal(t, want.ID, got.ID)
	require.Equal(t, want.UserID, got.UserID)
	require.Equal(t, want.RefreshToken, got.RefreshToken)
	require.Equal(t, want.UserAgent, got.UserAgent)
	require.Equal(t, want.ClientIp, got.ClientIp)
	require.Equal(t, want.IsBlocked, got.IsBlocked)
	require.WithinDuration(t, want.ExpiresAt, got.ExpiresAt, time.Second)
	require.WithinDuration(t, want.CreatedAt, got.CreatedAt, time.Second)
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
