package api

import (
	"bytes"
	"encoding/json"
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

	// Stubs.
	callGet := func(m *mockdb.MockStore, transferID int64) *gomock.Call {
		return m.EXPECT().GetTransfer(gomock.Any(), transferID)
	}

	// Table Testing
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
				requireBodyMatchTransfer(t, recorder.Body, transfer)
			},
		},
	}

	// Get Entry run test cases.
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

}

func randomTransfer() db.Transfer {
	return db.Transfer{
		ID:            util.RandomInt(1, 1000),
		FromAccountID: util.RandomInt(1, 1000),
		ToAccountID:   util.RandomInt(1, 1000),
		Amount:        util.RandomAmount(),
	}
}

func requireBodyMatchTransfer(t *testing.T, body *bytes.Buffer, want db.Transfer) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var got db.Transfer
	err = json.Unmarshal(data, &got)
	require.NoError(t, err)
	require.Equal(t, want, got)
}
