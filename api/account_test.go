package api

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mockdb "github.com/jimxshaw/trivial-bank/db/mocks"
	db "github.com/jimxshaw/trivial-bank/db/sqlc"
	"github.com/jimxshaw/trivial-bank/util"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newServerMock(m *mockdb.MockStore) *Server {
	return NewServer(m)
}

func newStoreMock(t *testing.T) (func(), *mockdb.MockStore) {
	ctrl := gomock.NewController(t)
	finish := func() {
		ctrl.Finish()
	}
	store := mockdb.NewMockStore(ctrl)
	return finish, store
}

func TestAccountAPI(t *testing.T) {
	account := randomAccount()

	accounts := []db.Account{
		randomAccount(),
		randomAccount(),
	}

	// Stubs.
	callGet := func(m *mockdb.MockStore, accountID int64) *gomock.Call {
		return m.EXPECT().GetAccount(gomock.Any(), accountID)
	}

	callList := func(m *mockdb.MockStore, params db.ListAccountsParams) *gomock.Call {
		return m.EXPECT().ListAccounts(gomock.Any(), params)
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
			Limit:  5,
			Offset: 0,
		}

		t.Run("happy path", func(t *testing.T) {
			finish, m := newStoreMock(t)
			defer finish()

			callList(m, params).
				Times(1).
				Return(accounts, nil)

			server := newServerMock(m)
			recorder := httptest.NewRecorder()

			request, err := http.NewRequest(method, url, nil)
			require.NoError(t, err)

			query := request.URL.Query()
			query.Add("page_id", "1")
			query.Add("page_size", "5")
			request.URL.RawQuery = query.Encode()

			server.router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusOK, recorder.Code)
		})

		t.Run("some error happened", func(t *testing.T) {
			finish, m := newStoreMock(t)
			defer finish()

			callList(m, params).
				Times(1).
				Return([]db.Account{}, errors.New("some error"))

			server := newServerMock(m)
			recorder := httptest.NewRecorder()

			request, err := http.NewRequest(method, url, nil)
			require.NoError(t, err)

			query := request.URL.Query()
			query.Add("page_id", "1")
			query.Add("page_size", "5")
			request.URL.RawQuery = query.Encode()

			server.router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusInternalServerError, recorder.Code)
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

			server := newServerMock(m)
			recorder := httptest.NewRecorder()

			request, err := http.NewRequest(method, url, nil)
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusOK, recorder.Code)
		})

		t.Run("some error happened", func(t *testing.T) {
			finish, m := newStoreMock(t)
			defer finish()

			callGet(m, account.ID).
				Times(1).
				Return(db.Account{}, errors.New("some error"))

			server := newServerMock(m)
			recorder := httptest.NewRecorder()

			request, err := http.NewRequest(method, url, nil)
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusInternalServerError, recorder.Code)
		})
	})

	// TODO: Create Account.

	// Update Account.
	t.Run("update account", func(t *testing.T) {
		accountToUpdate := db.Account{
			ID:        0,
			Owner:     "Darth Vader",
			Balance:   0,
			Currency:  "USD",
			CreatedAt: time.Date(1977, 5, 4, 0, 0, 0, 0, time.Local),
		}

		url := fmt.Sprintf("/accounts/%d", accountToUpdate.ID)
		method := http.MethodPut

		jsonStr := []byte(`{"owner":"Han Solo"}`)

		params := db.UpdateAccountParams{
			Owner: "Han Solo",
		}

		t.Run("happy path", func(t *testing.T) {
			finish, m := newStoreMock(t)
			defer finish()

			accountToUpdate.Owner = "Han Solo"

			callUpdate(m, params).
				Times(1).
				Return(accountToUpdate, nil)

			server := newServerMock(m)
			recorder := httptest.NewRecorder()

			request, err := http.NewRequest(method, url, bytes.NewBuffer(jsonStr))
			require.NoError(t, err)

			request.Header.Set("Content-Type", "application/json")

			server.router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusOK, recorder.Code)
		})

		t.Run("some error happened", func(t *testing.T) {
			finish, m := newStoreMock(t)
			defer finish()

			callUpdate(m, params).
				Times(1).
				Return(db.Account{}, errors.New("some error"))

			server := newServerMock(m)
			recorder := httptest.NewRecorder()

			request, err := http.NewRequest(method, url, bytes.NewBuffer(jsonStr))
			require.NoError(t, err)

			request.Header.Set("Content-Type", "application/json")

			server.router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusInternalServerError, recorder.Code)
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

			server := newServerMock(m)
			recorder := httptest.NewRecorder()

			request, err := http.NewRequest(method, url, nil)
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusOK, recorder.Code)
		})

		t.Run("some error happened", func(t *testing.T) {
			finish, m := newStoreMock(t)
			defer finish()

			callDelete(m, account.ID).
				Times(1).
				Return(errors.New("some error"))

			server := newServerMock(m)
			recorder := httptest.NewRecorder()

			request, err := http.NewRequest(method, url, nil)
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusInternalServerError, recorder.Code)
		})
	})
}

func randomAccount() db.Account {
	return db.Account{
		ID:       util.RandomInt(1, 1000),
		Owner:    util.RandomOwner(),
		Balance:  util.RandomAmount(),
		Currency: util.RandomCurrency(),
	}
}
