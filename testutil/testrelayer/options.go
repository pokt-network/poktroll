package testrelayer

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/session"
)

func WithTempStoresDirectory(t *testing.T) relayer.RelayerSessionsManagerOption {
	tmpDirPattern := fmt.Sprintf("%s_smt_kvstore", t.Name())
	tmpStoresDir, err := os.MkdirTemp("", tmpDirPattern)
	require.NoError(t, err)

	// Delete all temporary files and directories created by the test on completion.
	t.Cleanup(func() { _ = os.RemoveAll(tmpStoresDir) })

	return session.WithStoresDirectory(tmpStoresDir)
}
