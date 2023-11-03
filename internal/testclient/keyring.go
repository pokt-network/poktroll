package testclient

import (
	"testing"

	cosmoshd "github.com/cosmos/cosmos-sdk/crypto/hd"
	cosmoskeyring "github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/stretchr/testify/require"
)

// NewKey creates a new Secp256k1 key and mnemonic for the given name within
// the provided keyring.
func NewKey(
	t *testing.T,
	name string,
	keyring cosmoskeyring.Keyring,
) (key *cosmoskeyring.Record, mnemonic string) {
	t.Helper()

	key, mnemonic, err := keyring.NewMnemonic(
		name,
		cosmoskeyring.English,
		"m/44'/118'/0'/0/0",
		cosmoskeyring.DefaultBIP39Passphrase,
		cosmoshd.Secp256k1,
	)
	require.NoError(t, err)
	require.NotNil(t, key)

	return key, mnemonic
}
