package api

import (
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	mockdb "github.com/jimxshaw/trivial-bank/db/mocks"
	"github.com/jimxshaw/trivial-bank/util"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestMain(m *testing.M) {
	// In test mode, Gin will not print
	// logs in order to keep output clean.
	gin.SetMode(gin.TestMode)

	os.Exit(m.Run())
}

func newServerMock(t *testing.T, m *mockdb.MockStore) *Server {
	c := newConfigMock()
	s, err := NewServer(m, c)
	require.NoError(t, err)
	return s
}

func newStoreMock(t *testing.T) (func(), *mockdb.MockStore) {
	ctrl := gomock.NewController(t)
	finish := func() {
		ctrl.Finish()
	}
	store := mockdb.NewMockStore(ctrl)
	return finish, store
}

func newConfigMock() util.Config {
	return util.Config{
		DBDriver:            util.RandomString(10),
		DBSource:            util.RandomString(10),
		ServerAddress:       util.RandomString(10),
		TokenSymmetricKey:   util.RandomString(32),
		AccessTokenDuration: time.Minute,
	}
}
