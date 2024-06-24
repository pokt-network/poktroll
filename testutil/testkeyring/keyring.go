//go:generate go run ./gen_accounts/gen.go ./gen_accounts/template.go

package testkeyring

import (
	"fmt"
	"testing"

	ring_secp256k1 "github.com/athanorlabs/go-dleq/secp256k1"
	ringtypes "github.com/athanorlabs/go-dleq/types"
	cosmoscrypto "github.com/cosmos/cosmos-sdk/crypto"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// CreatePreGeneratedKeyringAccounts uses the mnemonic from limit number of
// pre-generated accounts to populated the provided keyring, kr. It then returns
// the pre-generated accounts which were used.
//
// TODO_TECHDEBT: Returning a new PreGeneratedAccountIterator instead of
// the slice of accounts could be more idiomatic. It would only contain keys which
// are known to be in the keyring.
func CreatePreGeneratedKeyringAccounts(
	t *testing.T,
	kr keyring.Keyring,
	limit int,
) []*PreGeneratedAccount {
	t.Helper()

	accounts := make([]*PreGeneratedAccount, limit)
	for i := range accounts {
		preGeneratedAccount := MustPreGeneratedAccountAtIndex(uint32(i))

		uid := fmt.Sprintf("key-%d", i)
		_, err := kr.NewAccount(
			uid,
			preGeneratedAccount.Mnemonic,
			keyring.DefaultBIP39Passphrase,
			types.FullFundraiserPath,
			hd.Secp256k1,
		)
		assert.NoError(t, err)

		accounts[i] = preGeneratedAccount
	}

	return accounts[:limit]
}

// GetSigningKeyFromAddress retrieves the signing key associated with the given
// bech32 address from the provided keyring.
func GetSigningKeyFromAddress(t *testing.T, bech32 string, keyRing keyring.Keyring) ringtypes.Scalar {
	t.Helper()

	addr, err := cosmostypes.AccAddressFromBech32(bech32)
	require.NoError(t, err)

	armorPrivKey, err := keyRing.ExportPrivKeyArmorByAddress(addr, "")
	require.NoError(t, err)

	privKey, _, err := cosmoscrypto.UnarmorDecryptPrivKey(armorPrivKey, "")
	require.NoError(t, err)

	curve := ring_secp256k1.NewCurve()
	signingKey, err := curve.DecodeToScalar(privKey.Bytes())
	require.NoError(t, err)

	return signingKey
}
