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
	"github.com/jimxshaw/trivial-bank/authentication/token"
	mockdb "github.com/jimxshaw/trivial-bank/db/mocks"
	db "github.com/jimxshaw/trivial-bank/db/sqlc"
	"github.com/jimxshaw/trivial-bank/util"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestTransferAPI(t *testing.T) {
	transfer := randomTransfer()

	transfers := []db.Transfer{
		{
			ID:            1,
			FromAccountID: 1,
			ToAccountID:   2,
			Amount:        100,
		},
		{
			ID:            2,
			FromAccountID: 2,
			ToAccountID:   1,
			Amount:        200,
		},
	}

	fromAccount := db.Account{
		ID:       1,
		UserID:   1,
		Balance:  1000,
		Currency: "USD",
	}

	toAccount := db.Account{
		ID:       2,
		UserID:   2,
		Balance:  500,
		Currency: "USD",
	}

	transferAmount := int64(250)

	transferTxParams := db.TransferTxParams{
		FromAccountID: fromAccount.ID,
		ToAccountID:   toAccount.ID,
		Amount:        transferAmount,
	}

	transferTxResult := db.TransferTxResult{
		Transfer: db.Transfer{
			ID:            1,
			FromAccountID: 1,
			ToAccountID:   2,
			Amount:        transferAmount,
		},
		FromAccount: fromAccount,
		ToAccount:   toAccount,
		FromEntry: db.Entry{
			ID:        1,
			AccountID: 1,
			Amount:    -transferAmount, // must be negative from source
		},
		ToEntry: db.Entry{
			ID:        2,
			AccountID: 2,
			Amount:    transferAmount, // must be positive to destination
		},
	}

	// Stubs.
	callList := func(m *mockdb.MockStore, params db.ListTransfersParams) *gomock.Call {
		return m.EXPECT().ListTransfers(gomock.Any(), params)
	}

	callGet := func(m *mockdb.MockStore, transferID int64) *gomock.Call {
		return m.EXPECT().GetTransfer(gomock.Any(), transferID)
	}

	callCreate := func(m *mockdb.MockStore, params db.TransferTxParams) *gomock.Call {
		return m.EXPECT().TransferTx(gomock.Any(), params)
	}

	callGetAccount := func(m *mockdb.MockStore, accountID int64) *gomock.Call {
		return m.EXPECT().GetAccount(gomock.Any(), accountID)
	}

	// Table Testing
	// List Transfers test cases.
	testCasesListTransfers := []struct {
		name          string
		pageID        int32
		pageSize      int32
		setupAuth     func(t *testing.T, request *http.Request, tokenGenerator token.Generator)
		stubs         func(m *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:     "happy path",
			pageID:   1,
			pageSize: 5,
			setupAuth: func(t *testing.T, req *http.Request, tokenGenerator token.Generator) {
				addAuthorizationToTest(t, req, tokenGenerator, mw.AuthTypeBearer, fromAccount.UserID, time.Minute)
			},
			stubs: func(m *mockdb.MockStore) {
				params := db.ListTransfersParams{
					UserID: fromAccount.UserID,
					Limit:  5,
					Offset: 0,
				}

				callList(m, params).
					Times(1).
					Return(transfers, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatch(t, recorder.Body, transfers)
			},
		},
		{
			name:     "some error happened",
			pageID:   1,
			pageSize: 5,
			setupAuth: func(t *testing.T, req *http.Request, tokenGenerator token.Generator) {
				addAuthorizationToTest(t, req, tokenGenerator, mw.AuthTypeBearer, fromAccount.UserID, time.Minute)
			},
			stubs: func(m *mockdb.MockStore) {
				params := db.ListTransfersParams{
					UserID: fromAccount.UserID,
					Limit:  5,
					Offset: 0,
				}

				callList(m, params).
					Times(1).
					Return([]db.Transfer{}, errors.New("some error"))
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:     "invalid query parameters",
			pageID:   0,
			pageSize: 0,
			setupAuth: func(t *testing.T, req *http.Request, tokenGenerator token.Generator) {
				addAuthorizationToTest(t, req, tokenGenerator, mw.AuthTypeBearer, fromAccount.UserID, time.Minute)
			},
			stubs: func(m *mockdb.MockStore) {
				params := db.ListTransfersParams{
					UserID: fromAccount.UserID,
					Limit:  0,
					Offset: 0,
				}

				callList(m, params).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
	}

	// List Transfers run test cases
	for i := range testCasesListTransfers {
		tc := testCasesListTransfers[i]

		t.Run(tc.name, func(t *testing.T) {
			finish, m := newStoreMock(t)
			defer finish()

			tc.stubs(m)

			s := newServerMock(t, m)
			rec := httptest.NewRecorder()

			url := fmt.Sprintf("/transfers?page_id=%d&page_size=%d", tc.pageID, tc.pageSize)
			req, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			tc.setupAuth(t, req, s.tokenGenerator)
			s.router.ServeHTTP(rec, req)

			tc.checkResponse(t, rec)
		})
	}

	// Get Transfer test cases.
	testCasesGetTransfer := []struct {
		name          string
		transferID    int64
		setupAuth     func(t *testing.T, request *http.Request, tokenGenerator token.Generator)
		stubs         func(m *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:       "happy path",
			transferID: transfer.ID,
			setupAuth: func(t *testing.T, req *http.Request, tokenGenerator token.Generator) {
				addAuthorizationToTest(t, req, tokenGenerator, mw.AuthTypeBearer, fromAccount.UserID, time.Minute)
			},
			stubs: func(m *mockdb.MockStore) {
				callGet(m, transfer.ID).
					Times(1).
					Return(transfer, nil)

				callGetAccount(m, transfer.FromAccountID).
					Times(1).
					Return(fromAccount, nil)

				callGetAccount(m, transfer.ToAccountID).
					Times(1).
					Return(toAccount, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatch(t, recorder.Body, transfer)
			},
		},
		{
			name:       "error getting fromAccount",
			transferID: transfer.ID,
			setupAuth: func(t *testing.T, req *http.Request, tokenGenerator token.Generator) {
				addAuthorizationToTest(t, req, tokenGenerator, mw.AuthTypeBearer, fromAccount.UserID, time.Minute)
			},
			stubs: func(m *mockdb.MockStore) {
				callGet(m, transfer.ID).
					Times(1).
					Return(transfer, nil)

				callGetAccount(m, transfer.FromAccountID).
					Times(1).
					Return(db.Account{}, errors.New("some error"))

				callGetAccount(m, transfer.ToAccountID).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:       "error getting toAccount",
			transferID: transfer.ID,
			setupAuth: func(t *testing.T, req *http.Request, tokenGenerator token.Generator) {
				addAuthorizationToTest(t, req, tokenGenerator, mw.AuthTypeBearer, fromAccount.UserID, time.Minute)
			},
			stubs: func(m *mockdb.MockStore) {
				callGet(m, transfer.ID).
					Times(1).
					Return(transfer, nil)

				callGetAccount(m, transfer.FromAccountID).
					Times(1).
					Return(fromAccount, nil)

				callGetAccount(m, transfer.ToAccountID).
					Times(1).
					Return(db.Account{}, errors.New("some error"))

			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:       "unauthorized user",
			transferID: transfer.ID,
			setupAuth: func(t *testing.T, req *http.Request, tokenGenerator token.Generator) {
				unauthorizedUserID := fromAccount.UserID + 111111 // userID that's not the sender or receiver.
				addAuthorizationToTest(t, req, tokenGenerator, mw.AuthTypeBearer, unauthorizedUserID, time.Minute)
			},
			stubs: func(m *mockdb.MockStore) {
				callGet(m, transfer.ID).
					Times(1).
					Return(transfer, nil)

				callGetAccount(m, transfer.FromAccountID).
					Times(1).
					Return(fromAccount, nil)

				callGetAccount(m, transfer.ToAccountID).
					Times(1).
					Return(toAccount, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name:       "not found",
			transferID: transfer.ID,
			setupAuth: func(t *testing.T, req *http.Request, tokenGenerator token.Generator) {
				addAuthorizationToTest(t, req, tokenGenerator, mw.AuthTypeBearer, fromAccount.UserID, time.Minute)
			},
			stubs: func(m *mockdb.MockStore) {
				callGet(m, transfer.ID).
					Times(1).
					Return(db.Transfer{}, sql.ErrNoRows)

				callGetAccount(m, transfer.FromAccountID).
					Times(0)

				callGetAccount(m, transfer.ToAccountID).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:       "some error happened",
			transferID: transfer.ID,
			setupAuth: func(t *testing.T, req *http.Request, tokenGenerator token.Generator) {
				addAuthorizationToTest(t, req, tokenGenerator, mw.AuthTypeBearer, fromAccount.UserID, time.Minute)
			},
			stubs: func(m *mockdb.MockStore) {
				callGet(m, transfer.ID).
					Times(1).
					Return(db.Transfer{}, errors.New("some error"))

				callGetAccount(m, transfer.FromAccountID).
					Times(0)

				callGetAccount(m, transfer.ToAccountID).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:       "invalid ID",
			transferID: 0,
			setupAuth: func(t *testing.T, req *http.Request, tokenGenerator token.Generator) {
				addAuthorizationToTest(t, req, tokenGenerator, mw.AuthTypeBearer, fromAccount.UserID, time.Minute)
			},
			stubs: func(m *mockdb.MockStore) {
				callGet(m, 0).
					Times(0)

				callGetAccount(m, transfer.FromAccountID).
					Times(0)

				callGetAccount(m, transfer.ToAccountID).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
	}

	// Get Transfer run test cases.
	for i := range testCasesGetTransfer {
		tc := testCasesGetTransfer[i]

		t.Run(tc.name, func(t *testing.T) {
			finish, m := newStoreMock(t)
			defer finish()

			tc.stubs(m)

			s := newServerMock(t, m)
			rec := httptest.NewRecorder()

			url := fmt.Sprintf("/transfers/%d", tc.transferID)
			req, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			tc.setupAuth(t, req, s.tokenGenerator)
			s.router.ServeHTTP(rec, req)

			tc.checkResponse(t, rec)
		})
	}

	// Create Transfer test cases.
	testCasesCreateTransfer := []struct {
		name          string
		body          []byte
		setupAuth     func(t *testing.T, req *http.Request, tokenGenerator token.Generator)
		stubs         func(m *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "happy path",
			body: []byte(`{"from_account_id":1,"to_account_id":2,"amount":250,"currency":"USD"}`),
			setupAuth: func(t *testing.T, req *http.Request, tokenGenerator token.Generator) {
				addAuthorizationToTest(t, req, tokenGenerator, mw.AuthTypeBearer, fromAccount.UserID, time.Minute)
			},
			stubs: func(m *mockdb.MockStore) {
				callGetAccount(m, fromAccount.ID).
					Times(1).
					Return(fromAccount, nil)

				callGetAccount(m, toAccount.ID).
					Times(1).
					Return(toAccount, nil)

				callCreate(m, transferTxParams).
					Times(1).
					Return(transferTxResult, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatch(t, recorder.Body, transferTxResult)
			},
		},
		{
			name: "unauthorized user",
			body: []byte(`{"from_account_id":1,"to_account_id":2,"amount":250,"currency":"USD"}`),
			setupAuth: func(t *testing.T, req *http.Request, tokenGenerator token.Generator) {
				unauthorizedUserID := fromAccount.UserID + 111111 // userID that's not the sender.
				addAuthorizationToTest(t, req, tokenGenerator, mw.AuthTypeBearer, unauthorizedUserID, time.Minute)
			},
			stubs: func(m *mockdb.MockStore) {
				callGetAccount(m, fromAccount.ID).
					Times(1).
					Return(fromAccount, nil)

				callGetAccount(m, toAccount.ID).
					Times(0)

				callCreate(m, transferTxParams).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "some error happened",
			body: []byte(`{"from_account_id":1,"to_account_id":2,"amount":250,"currency":"USD"}`),
			setupAuth: func(t *testing.T, req *http.Request, tokenGenerator token.Generator) {
				addAuthorizationToTest(t, req, tokenGenerator, mw.AuthTypeBearer, fromAccount.UserID, time.Minute)
			},
			stubs: func(m *mockdb.MockStore) {
				callGetAccount(m, fromAccount.ID).
					Times(1).
					Return(fromAccount, nil)

				callGetAccount(m, toAccount.ID).
					Times(1).
					Return(toAccount, nil)

				callCreate(m, transferTxParams).
					Times(1).
					Return(db.TransferTxResult{}, errors.New("some error"))
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "invalid body",
			body: []byte(`{"amount":250}`),
			setupAuth: func(t *testing.T, req *http.Request, tokenGenerator token.Generator) {
				addAuthorizationToTest(t, req, tokenGenerator, mw.AuthTypeBearer, fromAccount.UserID, time.Minute)
			},
			stubs: func(m *mockdb.MockStore) {
				callGetAccount(m, fromAccount.ID).
					Times(0)

				callGetAccount(m, toAccount.ID).
					Times(0)

				callCreate(m, transferTxParams).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "fromAccount ID does not exist",
			body: []byte(`{"from_account_id":1,"to_account_id":2,"amount":250,"currency":"USD"}`),
			setupAuth: func(t *testing.T, req *http.Request, tokenGenerator token.Generator) {
				addAuthorizationToTest(t, req, tokenGenerator, mw.AuthTypeBearer, fromAccount.UserID, time.Minute)
			},
			stubs: func(m *mockdb.MockStore) {
				callGetAccount(m, fromAccount.ID).
					Times(1).
					Return(db.Account{}, sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "fromAccount ID currency mismatch",
			body: []byte(`{"from_account_id":1,"to_account_id":2,"amount":250,"currency":"EUR"}`), // Euro
			setupAuth: func(t *testing.T, req *http.Request, tokenGenerator token.Generator) {
				addAuthorizationToTest(t, req, tokenGenerator, mw.AuthTypeBearer, fromAccount.UserID, time.Minute)
			},
			stubs: func(m *mockdb.MockStore) {
				callGetAccount(m, fromAccount.ID).
					Times(1).
					Return(fromAccount, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "toAccount ID does not exist",
			body: []byte(`{"from_account_id":1,"to_account_id":2,"amount":250,"currency":"USD"}`),
			setupAuth: func(t *testing.T, req *http.Request, tokenGenerator token.Generator) {
				addAuthorizationToTest(t, req, tokenGenerator, mw.AuthTypeBearer, fromAccount.UserID, time.Minute)
			},
			stubs: func(m *mockdb.MockStore) {
				callGetAccount(m, fromAccount.ID).
					Times(1).
					Return(fromAccount, nil)

				callGetAccount(m, toAccount.ID).
					Times(1).
					Return(db.Account{}, sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "toAccount ID currency mismatch",
			body: []byte(`{"from_account_id":1,"to_account_id":2,"amount":250,"currency":"USD"}`),
			setupAuth: func(t *testing.T, req *http.Request, tokenGenerator token.Generator) {
				addAuthorizationToTest(t, req, tokenGenerator, mw.AuthTypeBearer, fromAccount.UserID, time.Minute)
			},
			stubs: func(m *mockdb.MockStore) {
				callGetAccount(m, fromAccount.ID).
					Times(1).
					Return(fromAccount, nil)

				toAccount.Currency = "EUR"

				callGetAccount(m, toAccount.ID).
					Times(1).
					Return(toAccount, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "error during get account",
			body: []byte(`{"from_account_id":1,"to_account_id":2,"amount":250,"currency":"USD"}`),
			setupAuth: func(t *testing.T, req *http.Request, tokenGenerator token.Generator) {
				addAuthorizationToTest(t, req, tokenGenerator, mw.AuthTypeBearer, fromAccount.UserID, time.Minute)
			},
			stubs: func(m *mockdb.MockStore) {
				callGetAccount(m, fromAccount.ID).
					Times(1).
					Return(db.Account{}, errors.New("some error"))
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
	}

	// Create Transfer run test cases.
	for i := range testCasesCreateTransfer {
		tc := testCasesCreateTransfer[i]

		t.Run(tc.name, func(t *testing.T) {
			finish, m := newStoreMock(t)
			defer finish()

			tc.stubs(m)

			s := newServerMock(t, m)
			rec := httptest.NewRecorder()

			req, err := http.NewRequest(http.MethodPost, "/transfers", bytes.NewBuffer(tc.body))
			require.NoError(t, err)

			tc.setupAuth(t, req, s.tokenGenerator)
			s.router.ServeHTTP(rec, req)

			tc.checkResponse(t, rec)
		})
	}

}

func randomTransfer() db.Transfer {
	return db.Transfer{
		ID:            util.RandomInt(1, 1000),
		FromAccountID: util.RandomInt(1, 1000),
		ToAccountID:   util.RandomInt(1, 1000),
		Amount:        util.RandomAmount(),
	}
}

func requireBodyMatch(t *testing.T, body *bytes.Buffer, want interface{}) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	switch v := want.(type) {
	case db.Transfer:
		var got db.Transfer
		err = json.Unmarshal(data, &got)
		require.NoError(t, err)
		require.Equal(t, v, got)
	case []db.Transfer:
		var got []db.Transfer
		err = json.Unmarshal(data, &got)
		require.NoError(t, err)
		require.Equal(t, v, got)
	case db.TransferTxResult:
		var got db.TransferTxResult
		err = json.Unmarshal(data, &got)
		require.NoError(t, err)
		require.Equal(t, v, got)
	default:
		t.Fatalf("unsupported type: %T", want)
	}
}
