package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	mockdb "github.com/jimxshaw/trivial-bank/db/mocks"
	db "github.com/jimxshaw/trivial-bank/db/sqlc"
	"github.com/jimxshaw/trivial-bank/util"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestUserAPI(t *testing.T) {
	user, password := randomUser(t)

	// Stubs.
	callCreate := func(m *mockdb.MockStore, params db.CreateUserParams) *gomock.Call {
		return m.EXPECT().CreateUser(gomock.Any(), params)
	}

	// Create User test cases.
	testCasesCreateUser := []struct {
		name          string
		body          gin.H
		stubs         func(m *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "happy path",
			body: gin.H{
				"first_name": user.FirstName,
				"last_name":  user.LastName,
				"email":      user.Email,
				"username":   user.Username,
				"password":   password,
			},
			stubs: func(m *mockdb.MockStore) {
				params := db.CreateUserParams{
					FirstName: user.FirstName,
					LastName:  user.LastName,
					Email:     user.Email,
					Username:  user.Username,
					Password:  user.Password,
				}

				callCreate(m, params).
					Times(1).
					Return(user, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchUser(t, recorder.Body, user)
			},
		},
	}

	// Create User run test cases.
	for i := range testCasesCreateUser {
		tc := testCasesCreateUser[i]

		t.Run(tc.name, func(t *testing.T) {
			finish, m := newStoreMock(t)
			defer finish()

			tc.stubs(m)

			server := newServerMock(m)
			recorder := httptest.NewRecorder()

			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			url := "/users"
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)

			tc.checkResponse(t, recorder)
		})
	}

}

func randomUser(t *testing.T) (user db.User, password string) {
	password = fmt.Sprintf("%s%s", util.RandomString(10), "ABC123@")
	hash, err := util.HashPassword(password)
	require.NoError(t, err)

	username := fmt.Sprintf("%s%d", util.RandomString(8), util.RandomInt(1, 100))

	user = db.User{
		ID:        util.RandomInt(1, 10_000),
		FirstName: util.RandomString(10),
		LastName:  util.RandomString(10),
		Email:     util.RandomEmail(),
		Username:  username,
		Password:  hash,
	}
	return
}

func requireBodyMatchUser(t *testing.T, body *bytes.Buffer, want db.User) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var got db.User
	err = json.Unmarshal(data, &got)

	require.NoError(t, err)
	require.Equal(t, want.FirstName, got.FirstName)
	require.Equal(t, want.LastName, got.LastName)
	require.Equal(t, want.Username, got.Username)
	require.Equal(t, want.Email, got.Email)
	require.Empty(t, got.Password)
}
