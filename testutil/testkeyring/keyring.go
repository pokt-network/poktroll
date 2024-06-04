//go:generate go run ./gen_accounts/gen.go ./gen_accounts/template.go

package testkeyring

import (
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
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
