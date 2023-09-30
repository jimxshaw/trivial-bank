package api

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"

	db "github.com/jimxshaw/trivial-bank/db/sqlc"
	"github.com/jimxshaw/trivial-bank/util"
	"github.com/stretchr/testify/require"
)

func TestTransferAPI(t *testing.T) {

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
