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

	mockdb "github.com/jimxshaw/trivial-bank/db/mocks"
	db "github.com/jimxshaw/trivial-bank/db/sqlc"
	"github.com/jimxshaw/trivial-bank/util"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestTransferAPI(t *testing.T) {
	transfer := randomTransfer()

	transfers := []db.Transfer{
		randomTransfer(),
		randomTransfer(),
	}

	fromAccount := db.Account{
		ID:       1,
		Owner:    "Bilbo",
		Balance:  1000,
		Currency: "USD",
	}

	toAccount := db.Account{
		ID:       2,
		Owner:    "Thorin",
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
		stubs         func(m *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:     "happy path",
			pageID:   1,
			pageSize: 5,
			stubs: func(m *mockdb.MockStore) {
				params := db.ListTransfersParams{
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
			stubs: func(m *mockdb.MockStore) {
				params := db.ListTransfersParams{
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
			stubs: func(m *mockdb.MockStore) {
				params := db.ListTransfersParams{
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

			server := newServerMock(m)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/transfers?page_id=%d&page_size=%d", tc.pageID, tc.pageSize)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)

			tc.checkResponse(t, recorder)
		})
	}

	// Get Transfer test cases.
	testCasesGetTransfer := []struct {
		name          string
		transferID    int64
		stubs         func(m *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:       "happy path",
			transferID: transfer.ID,
			stubs: func(m *mockdb.MockStore) {
				callGet(m, transfer.ID).
					Times(1).
					Return(transfer, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatch(t, recorder.Body, transfer)
			},
		},
		{
			name:       "not found",
			transferID: transfer.ID,
			stubs: func(m *mockdb.MockStore) {
				callGet(m, transfer.ID).
					Times(1).
					Return(db.Transfer{}, sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:       "some error happened",
			transferID: transfer.ID,
			stubs: func(m *mockdb.MockStore) {
				callGet(m, transfer.ID).
					Times(1).
					Return(db.Transfer{}, errors.New("some error"))
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:       "invalid ID",
			transferID: 0,
			stubs: func(m *mockdb.MockStore) {
				callGet(m, 0).
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

			server := newServerMock(m)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/transfers/%d", tc.transferID)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)

			tc.checkResponse(t, recorder)
		})
	}

	// Create Transfer test cases.
	testCasesCreateTransfer := []struct {
		name          string
		body          []byte
		stubs         func(m *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "happy path",
			body: []byte(`{"from_account_id":1,"to_account_id":2,"amount":250,"currency":"USD"}`),
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
	}

	// Create Transfer run test cases.
	for i := range testCasesCreateTransfer {
		tc := testCasesCreateTransfer[i]

		t.Run(tc.name, func(t *testing.T) {
			finish, m := newStoreMock(t)
			defer finish()

			tc.stubs(m)

			server := newServerMock(m)
			recorder := httptest.NewRecorder()

			request, err := http.NewRequest(http.MethodPost, "/transfers", bytes.NewBuffer(tc.body))
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)

			tc.checkResponse(t, recorder)
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
