package api

import (
	"testing"

	"github.com/ElrondNetwork/elrond-eth-bridge/api/errors"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/stretchr/testify/require"
)

func TestNewHttpServer_NilServerShouldErr(t *testing.T) {
	t.Parallel()

	hs, err := NewHttpServer(nil)
	require.Equal(t, errors.ErrNilHttpServer, err)
	require.True(t, check.IfNil(hs))
}
