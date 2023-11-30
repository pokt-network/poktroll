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

func CreatePreGeneratedKeyringAccounts(
	t *testing.T,
	kr keyring.Keyring,
	limit int,
) []*PreGeneratedAccount {
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

	return accounts
}
