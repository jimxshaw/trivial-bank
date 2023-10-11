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

func TestEntryAPI(t *testing.T) {
	entry := randomEntry()

	entries := []db.Entry{
		randomEntry(),
		randomEntry(),
	}

	// Stubs.
	callList := func(m *mockdb.MockStore, params db.ListEntriesParams) *gomock.Call {
		return m.EXPECT().ListEntries(gomock.Any(), params)
	}

	callGet := func(m *mockdb.MockStore, entryID int64) *gomock.Call {
		return m.EXPECT().GetEntry(gomock.Any(), entryID)
	}

	// Table Testing
	// List Entries test cases.
	testCasesListEntries := []struct {
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
				params := db.ListEntriesParams{
					Limit:  5,
					Offset: 0,
				}

				callList(m, params).
					Times(1).
					Return(entries, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name:     "some error happened",
			pageID:   1,
			pageSize: 5,
			stubs: func(m *mockdb.MockStore) {
				params := db.ListEntriesParams{
					Limit:  5,
					Offset: 0,
				}

				callList(m, params).
					Times(1).
					Return([]db.Entry{}, errors.New("some error"))
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
				params := db.ListEntriesParams{
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

	// List Entries run test cases
	for i := range testCasesListEntries {
		tc := testCasesListEntries[i]

		t.Run(tc.name, func(t *testing.T) {
			finish, m := newStoreMock(t)
			defer finish()

			tc.stubs(m)

			s := newServerMock(t, m)
			rec := httptest.NewRecorder()

			url := fmt.Sprintf("/entries?page_id=%d&page_size=%d", tc.pageID, tc.pageSize)
			req, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			s.router.ServeHTTP(rec, req)

			tc.checkResponse(t, rec)
		})
	}

	// Get Entry define test cases.
	testCasesGetEntry := []struct {
		name          string
		entryID       int64
		stubs         func(m *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:    "happy path",
			entryID: entry.ID,
			stubs: func(m *mockdb.MockStore) {
				callGet(m, entry.ID).
					Times(1).
					Return(entry, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchEntry(t, recorder.Body, entry)
			},
		},
		{
			name:    "not found",
			entryID: entry.ID,
			stubs: func(m *mockdb.MockStore) {
				callGet(m, entry.ID).
					Times(1).
					Return(db.Entry{}, sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:    "some error happened",
			entryID: entry.ID,
			stubs: func(m *mockdb.MockStore) {
				callGet(m, entry.ID).
					Times(1).
					Return(db.Entry{}, errors.New("some error"))
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:    "invalid ID",
			entryID: 0,
			stubs: func(m *mockdb.MockStore) {
				callGet(m, 0).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
	}

	// Get Entry run test cases.
	for i := range testCasesGetEntry {
		tc := testCasesGetEntry[i]

		t.Run(tc.name, func(t *testing.T) {
			finish, m := newStoreMock(t)
			defer finish()

			tc.stubs(m)

			s := newServerMock(t, m)
			rec := httptest.NewRecorder()

			url := fmt.Sprintf("/entries/%d", tc.entryID)
			req, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			s.router.ServeHTTP(rec, req)

			tc.checkResponse(t, rec)
		})
	}

}

func randomEntry() db.Entry {
	return db.Entry{
		ID:        util.RandomInt(1, 1000),
		AccountID: util.RandomInt(1, 1000),
		Amount:    util.RandomAmount(),
	}
}

func requireBodyMatchEntry(t *testing.T, body *bytes.Buffer, want db.Entry) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var got db.Entry
	err = json.Unmarshal(data, &got)
	require.NoError(t, err)
	require.Equal(t, want, got)
}
