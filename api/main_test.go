package api

import (
	"os"
	"testing"
	"time"

	mockdb "github.com/andyrestart9/bank/db/mock"
	db "github.com/andyrestart9/bank/db/sqlc"
	"github.com/andyrestart9/bank/util"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T, store db.Store) *Server {
	config := util.Config{
		TokenSymmetricKey:   util.RandomString(32, false),
		AccessTokenDuration: time.Minute,
	}
	server, err := NewServer(config, store)
	require.NoError(t, err)
	return server
}

func TestMain(m *testing.M) {
	var _ db.Store = (*mockdb.MockStore)(nil) // 確認 mockdb.MockStore 實現了 db.Store interface

	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}
