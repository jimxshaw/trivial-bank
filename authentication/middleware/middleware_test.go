package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jimxshaw/trivial-bank/authentication/token"
	"github.com/stretchr/testify/require"
)

// MockTokenGenerator is a mock implementation of the Generator interface
type MockTokenGenerator struct {
	generatedToken  string
	validatePayload *token.Payload
	validateError   error
}

func NewMockTokenGenerator() *MockTokenGenerator {
	return &MockTokenGenerator{}
}

// WithGeneratedToken allows you to set the mock's return value for GenerateToken
func (m *MockTokenGenerator) WithGeneratedToken(token string) *MockTokenGenerator {
	m.generatedToken = token
	return m
}

// WithValidatedPayload allows you to set the mock's return value for ValidateToken
func (m *MockTokenGenerator) WithValidatedPayload(payload *token.Payload) *MockTokenGenerator {
	m.validatePayload = payload
	return m
}

// WithValidateError allows you to set the mock's error return for ValidateToken
func (m *MockTokenGenerator) WithValidateError() *MockTokenGenerator {
	m.validateError = errors.New("mocked validate token error")
	return m
}

func (m *MockTokenGenerator) GenerateToken(userID int64, duration time.Duration) (string, *token.Payload, error) {
	return m.generatedToken, m.validatePayload, nil
}

func (m *MockTokenGenerator) ValidateToken(token string) (*token.Payload, error) {
	if m.validateError != nil {
		return nil, m.validateError
	}
	return m.validatePayload, nil
}

func TestAuthGinMiddleware(t *testing.T) {
	validPayload := &token.Payload{UserID: 123}

	testCasesAuthGinMiddleware := []struct {
		name               string
		authHeader         string
		mockTokenGenerator *MockTokenGenerator
		expectedStatusCode int
		expectedResponse   string
	}{
		{
			name:               "no authorization header",
			authHeader:         "",
			expectedStatusCode: http.StatusUnauthorized,
			expectedResponse:   "authorization header is missing",
		},
		{
			name:               "invalid authorization header format",
			authHeader:         "Bearer",
			expectedStatusCode: http.StatusUnauthorized,
			expectedResponse:   "invalid authorization header format",
		},
		{
			name:               "unsupported authorization type",
			authHeader:         "Unsupported abcdefg",
			expectedStatusCode: http.StatusUnauthorized,
			expectedResponse:   "unsupported authorization type unsupported",
		},
		{
			name:               "valid token",
			authHeader:         "Bearer validtoken",
			mockTokenGenerator: NewMockTokenGenerator().WithGeneratedToken("validtoken").WithValidatedPayload(validPayload),
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "invalid token",
			authHeader:         "Bearer invalidtoken",
			mockTokenGenerator: NewMockTokenGenerator().WithValidateError(),
			expectedStatusCode: http.StatusUnauthorized,
			expectedResponse:   "mocked validate token error",
		},
	}

	for i := range testCasesAuthGinMiddleware {
		tc := testCasesAuthGinMiddleware[i]

		t.Run(tc.name, func(t *testing.T) {
			r := gin.Default()

			r.Use(AuthGinMiddleware(tc.mockTokenGenerator))

			r.GET("/test", func(ctx *gin.Context) {
				payload, _ := ctx.Get(string(AuthPayloadKey))
				if payload != nil {
					ctx.String(http.StatusOK, "OK")
				}
			})

			req, err := http.NewRequest(http.MethodGet, "/test", nil)
			require.NoError(t, err)
			req.Header.Set(AuthHeaderKey, tc.authHeader)

			res := httptest.NewRecorder()
			r.ServeHTTP(res, req)

			require.Equal(t, tc.expectedStatusCode, res.Code)

			if tc.expectedResponse != "" {
				require.Contains(t, res.Body.String(), tc.expectedResponse)
			}
		})
	}
}
