package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	mockdb "github.com/jimxshaw/trivial-bank/db/mocks"
	db "github.com/jimxshaw/trivial-bank/db/sqlc"
	"github.com/jimxshaw/trivial-bank/util"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

/* Custom Matcher */
// Using bcrypt, even if we hash the same password multiple times,
// the output hash will be different every time due to the random salt.
// This custom matcher (based on gomock's Matcher interface) resolves
// mismatched hashed passwords during testing.
type eqCreateUserParamsMatcher struct {
	params   db.CreateUserParams
	password string // raw password
}

func (e eqCreateUserParamsMatcher) Matches(x interface{}) bool {
	params, ok := x.(db.CreateUserParams)
	if !ok {
		return false
	}

	err := util.ComparePasswords(e.password, params.Password)
	if err != nil {
		return false
	}

	e.params.Password = params.Password

	return reflect.DeepEqual(e.params, params)
}

func (e eqCreateUserParamsMatcher) String() string {
	return fmt.Sprintf("matches params %v and password %v", e.params, e.password)
}

func EqCreateUserParams(params db.CreateUserParams, password string) gomock.Matcher {
	return eqCreateUserParamsMatcher{params, password}
}

/* Unit Tests */
func TestUserAPI(t *testing.T) {
	user, password := randomUser(t)

	// Stubs.
	callCreate := func(m *mockdb.MockStore, params db.CreateUserParams, password string) *gomock.Call {
		return m.EXPECT().CreateUser(gomock.Any(), EqCreateUserParams(params, password))
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
				}

				callCreate(m, params, password).
					Times(1).
					Return(user, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchUser(t, recorder.Body, user)
			},
		},
		{
			name: "some error happened",
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

				callCreate(m, params, password).
					Times(1).
					Return(db.User{}, errors.New("some error"))
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "invalid body",
			body: gin.H{
				"first_name": "",
				"last_name":  "",
				"email":      "",
				"username":   "",
				"password":   "",
			},
			stubs: func(m *mockdb.MockStore) {
				callCreate(m, db.CreateUserParams{}, "").
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
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
