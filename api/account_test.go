package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mw "github.com/jimxshaw/trivial-bank/authentication/middleware"
	mockdb "github.com/jimxshaw/trivial-bank/db/mocks"
	db "github.com/jimxshaw/trivial-bank/db/sqlc"
	"github.com/jimxshaw/trivial-bank/util"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestAccountAPI(t *testing.T) {
	user, _ := randomUser(t)
	account := randomAccount(user.ID)

	accounts := []db.Account{
		randomAccount(user.ID),
		randomAccount(user.ID),
	}

	// Stubs.
	callGet := func(m *mockdb.MockStore, accountID int64) *gomock.Call {
		return m.EXPECT().GetAccount(gomock.Any(), accountID)
	}

	callList := func(m *mockdb.MockStore, params db.ListAccountsParams) *gomock.Call {
		return m.EXPECT().ListAccounts(gomock.Any(), params)
	}

	callCreate := func(m *mockdb.MockStore, params db.CreateAccountParams) *gomock.Call {
		return m.EXPECT().CreateAccount(gomock.Any(), params)
	}

	callUpdate := func(m *mockdb.MockStore, params db.UpdateAccountParams) *gomock.Call {
		return m.EXPECT().UpdateAccount(gomock.Any(), params)
	}

	callDelete := func(m *mockdb.MockStore, accountID int64) *gomock.Call {
		return m.EXPECT().DeleteAccount(gomock.Any(), accountID)
	}

	// List Accounts.
	t.Run("list accounts", func(t *testing.T) {
		url := "/accounts"
		method := http.MethodGet

		params := db.ListAccountsParams{
			UserID: account.UserID,
			Limit:  5,
			Offset: 0,
		}

		t.Run("happy path", func(t *testing.T) {
			finish, m := newStoreMock(t)
			defer finish()

			callList(m, params).
				Times(1).
				Return(accounts, nil)

			s := newServerMock(t, m)
			rec := httptest.NewRecorder()

			req, err := http.NewRequest(method, url, nil)
			require.NoError(t, err)

			query := req.URL.Query()
			query.Add("page_id", "1")
			query.Add("page_size", "5")
			req.URL.RawQuery = query.Encode()

			addAuthorizationToTest(t, req, s.tokenGenerator, mw.AuthTypeBearer, account.UserID, time.Minute)
			s.router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusOK, rec.Code)
		})

		t.Run("no authorization", func(t *testing.T) {
			finish, m := newStoreMock(t)
			defer finish()

			callList(m, params).
				Times(0)

			s := newServerMock(t, m)
			rec := httptest.NewRecorder()

			req, err := http.NewRequest(method, url, nil)
			require.NoError(t, err)

			query := req.URL.Query()
			query.Add("page_id", "1")
			query.Add("page_size", "5")
			req.URL.RawQuery = query.Encode()

			s.router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusUnauthorized, rec.Code)
		})

		t.Run("some error happened", func(t *testing.T) {
			finish, m := newStoreMock(t)
			defer finish()

			callList(m, params).
				Times(1).
				Return([]db.Account{}, errors.New("some error"))

			s := newServerMock(t, m)
			rec := httptest.NewRecorder()

			req, err := http.NewRequest(method, url, nil)
			require.NoError(t, err)

			query := req.URL.Query()
			query.Add("page_id", "1")
			query.Add("page_size", "5")
			req.URL.RawQuery = query.Encode()

			addAuthorizationToTest(t, req, s.tokenGenerator, mw.AuthTypeBearer, account.UserID, time.Minute)
			s.router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusInternalServerError, rec.Code)
		})

		t.Run("invalid query parameters", func(t *testing.T) {
			finish, m := newStoreMock(t)
			defer finish()

			callList(m, db.ListAccountsParams{
				UserID: account.UserID,
				Limit:  0,
				Offset: 0}).
				Times(0)

			s := newServerMock(t, m)
			rec := httptest.NewRecorder()

			req, err := http.NewRequest(method, "/accounts?page_id=0&page_size=0", nil)
			require.NoError(t, err)

			addAuthorizationToTest(t, req, s.tokenGenerator, mw.AuthTypeBearer, account.UserID, time.Minute)
			s.router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusBadRequest, rec.Code)
		})
	})

	// Get Account.
	t.Run("get account", func(t *testing.T) {
		url := fmt.Sprintf("/accounts/%d", account.ID)
		method := http.MethodGet

		t.Run("happy path", func(t *testing.T) {
			finish, m := newStoreMock(t)
			defer finish()

			callGet(m, account.ID).
				Times(1).
				Return(account, nil)

			s := newServerMock(t, m)
			rec := httptest.NewRecorder()

			req, err := http.NewRequest(method, url, nil)
			require.NoError(t, err)

			addAuthorizationToTest(t, req, s.tokenGenerator, mw.AuthTypeBearer, account.UserID, time.Minute)
			s.router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusOK, rec.Code)
			requireBodyMatchAccount(t, rec.Body, account)
		})

		t.Run("no authorization", func(t *testing.T) {
			finish, m := newStoreMock(t)
			defer finish()

			callGet(m, account.ID).
				Times(0)

			s := newServerMock(t, m)
			rec := httptest.NewRecorder()

			req, err := http.NewRequest(method, url, nil)
			require.NoError(t, err)

			s.router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusUnauthorized, rec.Code)
		})

		t.Run("unauthorized user", func(t *testing.T) {
			finish, m := newStoreMock(t)
			defer finish()

			otherUserID := util.RandomInt(5000, 10000)
			otherAccount := db.Account{
				ID:       account.ID,
				UserID:   otherUserID, // Different user than the auth token.
				Balance:  2000,
				Currency: "USD",
			}

			callGet(m, otherAccount.ID).
				Times(1).
				Return(otherAccount, nil)

			s := newServerMock(t, m)
			rec := httptest.NewRecorder()

			req, err := http.NewRequest(method, url, nil)
			require.NoError(t, err)

			addAuthorizationToTest(t, req, s.tokenGenerator, mw.AuthTypeBearer, account.UserID, time.Minute)
			s.router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusUnauthorized, rec.Code)
		})

		t.Run("some error happened", func(t *testing.T) {
			finish, m := newStoreMock(t)
			defer finish()

			callGet(m, account.ID).
				Times(1).
				Return(db.Account{}, errors.New("some error"))

			s := newServerMock(t, m)
			rec := httptest.NewRecorder()

			req, err := http.NewRequest(method, url, nil)
			require.NoError(t, err)

			addAuthorizationToTest(t, req, s.tokenGenerator, mw.AuthTypeBearer, account.UserID, time.Minute)
			s.router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusInternalServerError, rec.Code)
			requireBodyMatchAccount(t, rec.Body, db.Account{})
		})

		t.Run("invalid ID", func(t *testing.T) {
			finish, m := newStoreMock(t)
			defer finish()

			invalidID := int64(0)

			callGet(m, invalidID).
				Times(0)

			s := newServerMock(t, m)
			rec := httptest.NewRecorder()

			req, err := http.NewRequest(method, fmt.Sprintf("/accounts/%d", invalidID), nil)
			require.NoError(t, err)

			addAuthorizationToTest(t, req, s.tokenGenerator, mw.AuthTypeBearer, account.UserID, time.Minute)
			s.router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusBadRequest, rec.Code)
		})

		t.Run("not found", func(t *testing.T) {
			finish, m := newStoreMock(t)
			defer finish()

			callGet(m, account.ID).
				Times(1).
				Return(db.Account{}, sql.ErrNoRows)

			s := newServerMock(t, m)
			rec := httptest.NewRecorder()

			req, err := http.NewRequest(method, url, nil)
			require.NoError(t, err)

			addAuthorizationToTest(t, req, s.tokenGenerator, mw.AuthTypeBearer, account.UserID, time.Minute)
			s.router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusNotFound, rec.Code)
		})
	})

	// Create Account.
	t.Run("create account", func(t *testing.T) {
		url := "/accounts"
		method := http.MethodPost

		jsonStr := []byte(fmt.Sprintf(`{"user_id":%d,"currency":"USD"}`, account.UserID))

		params := db.CreateAccountParams{
			UserID:   account.UserID,
			Balance:  0,
			Currency: "USD",
		}

		t.Run("happy path", func(t *testing.T) {
			finish, m := newStoreMock(t)
			defer finish()

			newAccount := db.Account{
				Balance:  0,
				UserID:   account.UserID,
				Currency: "USD",
			}

			callCreate(m, params).
				Times(1).
				Return(newAccount, nil)

			s := newServerMock(t, m)
			rec := httptest.NewRecorder()

			req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonStr))
			require.NoError(t, err)

			req.Header.Set("Content-Type", "application/json")

			addAuthorizationToTest(t, req, s.tokenGenerator, mw.AuthTypeBearer, account.UserID, time.Minute)
			s.router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusOK, rec.Code)
			requireBodyMatchAccount(t, rec.Body, newAccount)
		})

		t.Run("no authorization", func(t *testing.T) {
			finish, m := newStoreMock(t)
			defer finish()

			callCreate(m, params).
				Times(0)

			s := newServerMock(t, m)
			rec := httptest.NewRecorder()

			req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonStr))
			require.NoError(t, err)

			req.Header.Set("Content-Type", "application/json")

			s.router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusUnauthorized, rec.Code)
		})

		t.Run("foreign key violation", func(t *testing.T) {
			finish, m := newStoreMock(t)
			defer finish()

			params := db.CreateAccountParams{
				UserID:   account.UserID,
				Currency: "USD",
				Balance:  0,
			}

			dbErr := &pq.Error{
				Code: "23503", // Error code for "foreign_key_violation" in PostgreSQL.
			}

			callCreate(m, params).
				Times(1).
				Return(db.Account{}, dbErr)

			s := newServerMock(t, m)
			rec := httptest.NewRecorder()

			req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonStr))
			require.NoError(t, err)

			req.Header.Set("Content-Type", "application/json")

			addAuthorizationToTest(t, req, s.tokenGenerator, mw.AuthTypeBearer, account.UserID, time.Minute)
			s.router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusForbidden, rec.Code)
		})

		t.Run("unique violation", func(t *testing.T) {
			finish, m := newStoreMock(t)
			defer finish()

			params := db.CreateAccountParams{
				UserID:   account.UserID,
				Currency: "USD",
				Balance:  0,
			}

			dbErr := &pq.Error{
				Code: "23505", // Error code for "unique_violation" in PostgreSQL.
			}

			callCreate(m, params).
				Times(1).
				Return(db.Account{}, dbErr)

			s := newServerMock(t, m)
			rec := httptest.NewRecorder()

			req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonStr))
			require.NoError(t, err)

			req.Header.Set("Content-Type", "application/json")

			addAuthorizationToTest(t, req, s.tokenGenerator, mw.AuthTypeBearer, account.UserID, time.Minute)
			s.router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusForbidden, rec.Code)
		})

		t.Run("some error happened", func(t *testing.T) {
			finish, m := newStoreMock(t)
			defer finish()

			callCreate(m, params).
				Times(1).
				Return(db.Account{}, errors.New("some error"))

			s := newServerMock(t, m)
			rec := httptest.NewRecorder()

			req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonStr))
			require.NoError(t, err)

			req.Header.Set("Content-Type", "application/json")

			addAuthorizationToTest(t, req, s.tokenGenerator, mw.AuthTypeBearer, account.UserID, time.Minute)
			s.router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusInternalServerError, rec.Code)
			requireBodyMatchAccount(t, rec.Body, db.Account{})
		})

		t.Run("invalid JSON payload", func(t *testing.T) {
			finish, m := newStoreMock(t)
			defer finish()

			callCreate(m, db.CreateAccountParams{}).
				Times(0)

			s := newServerMock(t, m)
			rec := httptest.NewRecorder()

			req, err := http.NewRequest(method, url, bytes.NewBuffer([]byte(`{}`)))
			require.NoError(t, err)

			req.Header.Set("Content-Type", "application/json")

			addAuthorizationToTest(t, req, s.tokenGenerator, mw.AuthTypeBearer, account.UserID, time.Minute)
			s.router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusBadRequest, rec.Code)
		})
	})

	// Update Account.
	t.Run("update account", func(t *testing.T) {
		accountToUpdate := db.Account{
			ID:        0,
			UserID:    1,
			Balance:   0,
			Currency:  "USD",
			CreatedAt: time.Date(1977, 5, 4, 0, 0, 0, 0, time.UTC),
		}

		url := fmt.Sprintf("/accounts/%d", accountToUpdate.ID)
		method := http.MethodPut

		jsonStr := []byte(`{"user_id":2}`)

		params := db.UpdateAccountParams{
			UserID: 2,
		}

		t.Run("happy path", func(t *testing.T) {
			finish, m := newStoreMock(t)
			defer finish()

			accountToUpdate.UserID = 2

			callUpdate(m, params).
				Times(1).
				Return(accountToUpdate, nil)

			s := newServerMock(t, m)
			rec := httptest.NewRecorder()

			req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonStr))
			require.NoError(t, err)

			req.Header.Set("Content-Type", "application/json")

			addAuthorizationToTest(t, req, s.tokenGenerator, mw.AuthTypeBearer, account.UserID, time.Minute)
			s.router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusOK, rec.Code)
			requireBodyMatchAccount(t, rec.Body, accountToUpdate)
		})

		t.Run("no authorization", func(t *testing.T) {
			finish, m := newStoreMock(t)
			defer finish()

			callUpdate(m, params).
				Times(0)

			s := newServerMock(t, m)
			rec := httptest.NewRecorder()

			req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonStr))
			require.NoError(t, err)

			req.Header.Set("Content-Type", "application/json")

			s.router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusUnauthorized, rec.Code)
		})

		t.Run("some error happened", func(t *testing.T) {
			finish, m := newStoreMock(t)
			defer finish()

			callUpdate(m, params).
				Times(1).
				Return(db.Account{}, errors.New("some error"))

			s := newServerMock(t, m)
			rec := httptest.NewRecorder()

			req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonStr))
			require.NoError(t, err)

			req.Header.Set("Content-Type", "application/json")

			addAuthorizationToTest(t, req, s.tokenGenerator, mw.AuthTypeBearer, account.UserID, time.Minute)
			s.router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusInternalServerError, rec.Code)
			requireBodyMatchAccount(t, rec.Body, db.Account{})
		})

		t.Run("invalid ID", func(t *testing.T) {
			finish, m := newStoreMock(t)
			defer finish()

			callUpdate(m, params).
				Times(0)

			s := newServerMock(t, m)
			rec := httptest.NewRecorder()

			req, err := http.NewRequest(method, "/accounts/hello", bytes.NewBuffer(jsonStr))
			require.NoError(t, err)

			req.Header.Set("Content-Type", "application/json")

			addAuthorizationToTest(t, req, s.tokenGenerator, mw.AuthTypeBearer, account.UserID, time.Minute)
			s.router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusBadRequest, rec.Code)
		})

		t.Run("invalid JSON payload", func(t *testing.T) {
			finish, m := newStoreMock(t)
			defer finish()

			callUpdate(m, db.UpdateAccountParams{}).
				Times(0)

			s := newServerMock(t, m)
			rec := httptest.NewRecorder()

			req, err := http.NewRequest(method, url, bytes.NewBuffer([]byte(`{}`)))
			require.NoError(t, err)

			req.Header.Set("Content-Type", "application/json")

			addAuthorizationToTest(t, req, s.tokenGenerator, mw.AuthTypeBearer, account.UserID, time.Minute)
			s.router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusBadRequest, rec.Code)
		})
	})

	// Delete Account.
	t.Run("delete account", func(t *testing.T) {
		url := fmt.Sprintf("/accounts/%d", account.ID)
		method := http.MethodDelete

		t.Run("happy path", func(t *testing.T) {
			finish, m := newStoreMock(t)
			defer finish()

			callDelete(m, account.ID).
				Times(1).
				Return(nil)

			s := newServerMock(t, m)
			rec := httptest.NewRecorder()

			req, err := http.NewRequest(method, url, nil)
			require.NoError(t, err)

			addAuthorizationToTest(t, req, s.tokenGenerator, mw.AuthTypeBearer, account.UserID, time.Minute)
			s.router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusOK, rec.Code)
		})

		t.Run("no authorization", func(t *testing.T) {
			finish, m := newStoreMock(t)
			defer finish()

			callDelete(m, account.ID).
				Times(0)

			s := newServerMock(t, m)
			rec := httptest.NewRecorder()

			req, err := http.NewRequest(method, url, nil)
			require.NoError(t, err)

			s.router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusUnauthorized, rec.Code)
		})

		t.Run("some error happened", func(t *testing.T) {
			finish, m := newStoreMock(t)
			defer finish()

			callDelete(m, account.ID).
				Times(1).
				Return(errors.New("some error"))

			s := newServerMock(t, m)
			rec := httptest.NewRecorder()

			req, err := http.NewRequest(method, url, nil)
			require.NoError(t, err)

			addAuthorizationToTest(t, req, s.tokenGenerator, mw.AuthTypeBearer, account.UserID, time.Minute)
			s.router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusInternalServerError, rec.Code)
		})

		t.Run("invalid ID", func(t *testing.T) {
			finish, m := newStoreMock(t)
			defer finish()

			invalidID := int64(0)

			callDelete(m, invalidID).
				Times(0)

			s := newServerMock(t, m)
			rec := httptest.NewRecorder()

			req, err := http.NewRequest(method, fmt.Sprintf("/accounts/%d", invalidID), nil)
			require.NoError(t, err)

			addAuthorizationToTest(t, req, s.tokenGenerator, mw.AuthTypeBearer, account.UserID, time.Minute)
			s.router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusBadRequest, rec.Code)
		})
	})
}

func randomAccount(userID int64) db.Account {
	return db.Account{
		ID:       util.RandomInt(1, 1000),
		UserID:   userID,
		Balance:  util.RandomAmount(),
		Currency: util.RandomCurrency(),
	}
}

func requireBodyMatchAccount(t *testing.T, body *bytes.Buffer, want db.Account) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var got db.Account
	err = json.Unmarshal(data, &got)
	require.NoError(t, err)
	require.Equal(t, want, got)
}
