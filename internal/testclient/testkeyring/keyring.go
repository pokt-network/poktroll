package testkeyring

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"

	"pocket/internal/testclient"
)

// NewTestKeyringWithKey creates a new in-memory keyring with a test key
// with testSigningKeyName as its name.
func NewTestKeyringWithKey(t *testing.T, keyName string) (keyring.Keyring, *keyring.Record) {
	keyring := keyring.NewInMemory(testclient.EncodingConfig.Marshaler)
	key, _ := testclient.NewKey(t, keyName, keyring)
	return keyring, key
}
