package testkeyring

import (
	"testing"

	"cosmossdk.io/depinject"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app"
	"github.com/pokt-network/poktroll/testutil/testclient"
)

// NewTestKeyringWithKey creates a new in-memory keyring with a test key
// with testSigningKeyName as its name.
func NewTestKeyringWithKey(t *testing.T, keyName string) (keyring.Keyring, *keyring.Record) {
	var marshaler codec.Codec
	deps := depinject.Configs(
		depinject.Supply(log.NewTestLogger(t)),
		app.AppConfig(),
	)
	err := depinject.Inject(deps, &marshaler)
	require.NoError(t, err)

	keyring := keyring.NewInMemory(marshaler)
	key, _ := testclient.NewKey(t, keyName, keyring)
	return keyring, key
}
