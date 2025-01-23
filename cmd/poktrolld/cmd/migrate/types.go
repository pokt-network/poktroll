package migrate

import (
	"fmt"

	cosmosmath "cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/app/volatile"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

// TODO_IN_THIS_COMMIT: godoc...
type morseImportWorkspace struct {
	// TODO_IN_THIS_COMMIT: godoc...
	addressToIdx map[string]uint64
	accounts     []*migrationtypes.MorseAccount
}

// nextIdx returns the next index to be used when appending a new account to the accounts slice.
func (miw *morseImportWorkspace) nextIdx() uint64 {
	return uint64(len(miw.accounts))
}

// lastIdx returns the last index of the accounts slice.
func (miw *morseImportWorkspace) lastIdx() uint64 {
	return uint64(len(miw.accounts) - 1)
}

// hasAccount returns true if the given address is present in the accounts slice.
func (miw *morseImportWorkspace) hasAccount(addr string) bool {
	_, ok := miw.addressToIdx[addr]
	return ok
}

// TODO_IN_THIS_COMMIT: godoc...
func (miw *morseImportWorkspace) ensureAccount(
	addr string,
	exportAccount *migrationtypes.MorseAuthAccount,
) (accountIdx uint64, balance cosmostypes.Coin) {
	var ok bool
	balance = cosmostypes.NewCoin(volatile.DenomuPOKT, cosmosmath.ZeroInt())

	if accountIdx, ok = miw.addressToIdx[addr]; ok {
		accountIdx = accountIdx
		importAccount := miw.accounts[accountIdx]
		// TODO_IN_THIS_COMMIT: comment... SHOULD ONLY be one denom (upokt).
		if len(importAccount.Coins) != 0 {
			balance = importAccount.Coins[0]
		}
	} else {
		accountIdx = miw.nextIdx()
		importAccount := &migrationtypes.MorseAccount{
			Address: exportAccount.Value.Address,
			PubKey:  exportAccount.Value.PubKey,
			Coins:   cosmostypes.Coins{balance},
		}
		miw.accounts = append(miw.accounts, importAccount)
		miw.addressToIdx[addr] = accountIdx
	}

	return accountIdx, balance
}

// TODO_IN_THIS_COMMIT: godoc...
func (miw *morseImportWorkspace) addUpokt(addr string, amount cosmosmath.Int) error {
	importAccountIdx, hasAccountAddr := miw.addressToIdx[addr]
	if !hasAccountAddr {
		return fmt.Errorf("account %q not found", addr)
	}

	account := miw.accounts[importAccountIdx]
	account.Coins[0].Amount = account.Coins[0].Amount.Add(amount)

	return nil
}
