package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jimxshaw/trivial-bank/authentication/token"
	"github.com/jimxshaw/trivial-bank/util"
	"github.com/stretchr/testify/require"
)

type mockTokenGenerator struct {
	valid       bool
	payload     *token.Payload
	generateErr error
	validateErr error
}

func (m *mockTokenGenerator) GenerateToken(userID int64, duration time.Duration) (string, error) {
	if m.generateErr != nil {
		return "", m.generateErr
	}
	return "mock-token", nil
}

func (m *mockTokenGenerator) ValidateToken(token string) (*token.Payload, error) {
	if m.validateErr != nil {
		return nil, m.validateErr
	}
	if m.valid {
		return m.payload, nil
	}
	return nil, errors.New("invalid token")
}

func TestAuthMiddleware(t *testing.T) {
	payload := &token.Payload{
		ID:        uuid.New(),
		UserID:    util.RandomInt(1, 1000),
		IssuedAt:  time.Now(),
		ExpiredAt: time.Now().Add(time.Minute),
	}

	testCasesAuthMiddleware := []struct {
		name           string
		givenToken     string
		givenAuthType  string
		tokenGenerator token.Generator
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "happy path",
			givenToken:     "valid-token",
			givenAuthType:  authTypeBearer,
			tokenGenerator: &mockTokenGenerator{valid: true, payload: payload},
			expectedStatus: http.StatusOK,
			expectedBody:   "next-handler-response",
		},
		{
			name:           "missing authorization header",
			givenToken:     "",
			givenAuthType:  "",
			tokenGenerator: &mockTokenGenerator{valid: true},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "authorization header is missing\n",
		},
		{
			name:           "invalid header format",
			givenToken:     "only-token",
			givenAuthType:  "",
			tokenGenerator: &mockTokenGenerator{valid: true},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "invalid authorization header format\n",
		},
		{
			name:           "unsupported authorization type",
			givenToken:     "token",
			givenAuthType:  "custom",
			tokenGenerator: &mockTokenGenerator{valid: true},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "unsupported authorization type custom\n",
		},
		{
			name:           "invalid token",
			givenToken:     "invalid-token",
			givenAuthType:  authTypeBearer,
			tokenGenerator: &mockTokenGenerator{valid: false},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "invalid token\n",
		},
	}

	for i := range testCasesAuthMiddleware {
		tc := testCasesAuthMiddleware[i]

		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/", nil)
			require.NoError(t, err)

			if tc.givenToken != "" || tc.givenAuthType != "" {
				req.Header.Add(authHeaderKey, fmt.Sprintf("%s %s", tc.givenAuthType, tc.givenToken))
			}

			rr := httptest.NewRecorder()

			mw := AuthMiddleware(tc.tokenGenerator)

			// A "next" handler is needed to check if the request successfully passed the middleware.
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, "next-handler-response")
			})

			mw(nextHandler).ServeHTTP(rr, req)

			require.Equal(t, tc.expectedStatus, rr.Code)
			require.Equal(t, tc.expectedBody, rr.Body.String())
		})
	}
}
