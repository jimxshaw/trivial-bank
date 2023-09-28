package api

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

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

	// Stubs.
	callGet := func(m *mockdb.MockStore, accountID int64) *gomock.Call {
		return m.EXPECT().GetAccount(gomock.Any(), accountID)
	}

	// TODO: List Accounts.

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

	// TODO: Update Account.

	// TODO: Delete Account.
}

func randomAccount() db.Account {
	return db.Account{
		ID:       util.RandomInt(1, 1000),
		Owner:    util.RandomOwner(),
		Balance:  util.RandomAmount(),
		Currency: util.RandomCurrency(),
	}
}
