package migrate

import (
	"fmt"

	cosmosmath "cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/app/volatile"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

// newMorseImportWorkspace returns a new morseImportWorkspace with fields initialized to their zero values.
func newMorseImportWorkspace() *morseImportWorkspace {
	return &morseImportWorkspace{
		addressToIdx: make(map[string]uint64),
		accountState: &migrationtypes.MorseAccountState{
			Accounts: make([]*migrationtypes.MorseAccount, 0),
		},
		lastAccTotalBalance:       cosmosmath.ZeroInt(),
		lastAccTotalAppStake:      cosmosmath.ZeroInt(),
		lastAccTotalSupplierStake: cosmosmath.ZeroInt(),
	}
}

// morseImportWorkspace is a helper struct that is used to consolidate the Morse account balance,
// application stake, and supplier stake for each account as an entry in the resulting MorseAccountState.
type morseImportWorkspace struct {
	// addressToIdx is a map from the Shannon bech32 address to the index of the
	// corresponding MorseAccount in the accounts slice.
	addressToIdx map[string]uint64
	// accountState is the final MorseAccountState that will be imported into Shannon.
	// It includes a slice of MorseAccount objects which are populated, by transforming
	// the input MorseStateExport into the output MorseAccountState.
	accountState *migrationtypes.MorseAccountState

	// lastAccAccountIdx is the index at which the most recent accumulation/totaling
	// (of actor counts, balances, and stakes) was performed such that the next
	// accumulation/totaling operation may reuse previous accumulations values.
	lastAccAccountIdx uint64
	// lastAccTotalBalance is the most recently accumulated balances of all Morse
	// accounts which have been processed.
	lastAccTotalBalance cosmosmath.Int
	// lastAccTotalAppStake is the most recently accumulated application stakes of
	// all Morse accounts which have been processed.
	lastAccTotalAppStake cosmosmath.Int
	// lastAccTotalSupplierStake is the most recently accumulated supplier stakes of
	// all Morse accounts which have been processed.
	lastAccTotalSupplierStake cosmosmath.Int
	// numAccounts is the number of accounts that have been processed.
	numAccounts uint64
	// numApplications is the number of applications that have been processed.
	numApplications uint64
	// numSuppliers is the number of suppliers that have been processed.
	numSuppliers uint64
}

// nextIdx returns the next index to be used when appending a new account to the accounts slice.
func (miw *morseImportWorkspace) nextIdx() uint64 {
	return uint64(len(miw.accountState.Accounts))
}

// hasAccount returns true if the given address is present in the accounts slice.
func (miw *morseImportWorkspace) hasAccount(addr string) bool {
	_, ok := miw.addressToIdx[addr]
	return ok
}

// TODO_IN_THIS_COMMIT: godoc...
func (miw *morseImportWorkspace) debugLogProgress(accountIdx int) {
	totalBalance := miw.totalBalance()
	totalAppStake := miw.totalAppStake()
	totalSupplierStake := miw.totalSupplierStake()
	grandTotal := totalBalance.Add(totalAppStake).Add(totalSupplierStake)

	logger.Debug().
		Int("account_idx", accountIdx).
		Uint64("num_accounts", miw.numAccounts).
		Uint64("num_applications", miw.numApplications).
		Uint64("num_suppliers", miw.numSuppliers).
		Str("total_balance", totalBalance.String()).
		Str("total_app_stake", totalAppStake.String()).
		Str("total_supplier_stake", totalSupplierStake.String()).
		Str("grand_total", grandTotal.String()).
		Msg("processing accounts...")
}

// TODO_IN_THIS_COMMIT: godoc...
func (miw *morseImportWorkspace) infoLogComplete() error {
	accountStateHash, err := miw.accountState.GetHash()
	if err != nil {
		return err
	}

	logger.Info().
		Uint64("num_accounts", miw.numAccounts).
		Uint64("num_applications", miw.numApplications).
		Uint64("num_suppliers", miw.numSuppliers).
		Str("total_balance", miw.totalBalance().String()).
		Str("total_app_stake", miw.totalAppStake().String()).
		Str("total_supplier_stake", miw.totalSupplierStake().String()).
		Str("grand_total", miw.grandTotal().String()).
		Str("morse_account_state_hash", fmt.Sprintf("%x", accountStateHash)).
		Msg("processing accounts complete")
	return nil
}

// TODO_IN_THIS_COMMIT: godoc...
func (miw *morseImportWorkspace) totalBalance() cosmosmath.Int {
	miw.accumulateTotals()
	return miw.lastAccTotalBalance
}

// TODO_IN_THIS_COMMIT: godoc...
func (miw *morseImportWorkspace) totalAppStake() cosmosmath.Int {
	miw.accumulateTotals()
	return miw.lastAccTotalAppStake
}

// TODO_IN_THIS_COMMIT: godoc...
func (miw *morseImportWorkspace) totalSupplierStake() cosmosmath.Int {
	miw.accumulateTotals()
	return miw.lastAccTotalSupplierStake
}

// TODO_IN_THIS_COMMIT: godoc...
func (miw *morseImportWorkspace) grandTotal() cosmosmath.Int {
	return miw.totalBalance().
		Add(miw.totalAppStake()).
		Add(miw.totalSupplierStake())
}

// TODO_IN_THIS_COMMIT: godoc...
func (miw *morseImportWorkspace) accumulateTotals() {
	for idx, account := range miw.accountState.Accounts[miw.lastAccAccountIdx:] {
		miw.lastAccTotalBalance = miw.lastAccTotalBalance.Add(account.Coins[0].Amount)
		miw.lastAccTotalAppStake = miw.lastAccTotalAppStake.Add(account.Coins[0].Amount)
		miw.lastAccTotalSupplierStake = miw.lastAccTotalSupplierStake.Add(account.Coins[0].Amount)
		miw.lastAccAccountIdx = uint64(idx)
	}
}

// ensureAccount ensures that the given address is present in the accounts slice
// and that its corresponding address is in the addressToIdx map. If the address
// is not present, it is added to the accounts slice and the addressToIdx map.
func (miw *morseImportWorkspace) ensureAccount(
	addr string,
	exportAccount *migrationtypes.MorseAuthAccount,
) (accountIdx uint64, balance cosmostypes.Coin, err error) {
	var ok bool
	balance = cosmostypes.NewCoin(volatile.DenomuPOKT, cosmosmath.ZeroInt())

	if accountIdx, ok = miw.addressToIdx[addr]; ok {
		logger.Warn().Str("address", addr).Msg("unexpected workspace state: account already exists")

		importAccount := miw.accountState.Accounts[accountIdx]
		// Each account should have EXACTLY one token denomination.
		if len(importAccount.Coins) != 1 {
			err := ErrMorseStateTransform.Wrapf("account %q has multiple token denominations: %s", addr, importAccount.Coins)
			return 0, cosmostypes.Coin{}, err
		}
		balance = importAccount.Coins[0]
	} else {
		accountIdx = miw.nextIdx()
		importAccount := &migrationtypes.MorseAccount{
			Address: exportAccount.Value.Address,
			PubKey:  exportAccount.Value.PubKey,
			Coins:   cosmostypes.Coins{balance},
		}
		miw.accountState.Accounts = append(miw.accountState.Accounts, importAccount)
		miw.addressToIdx[addr] = accountIdx
	}

	return accountIdx, balance, nil
}

// addUpokt adds the given amount to the corresponding account balances in the morseWorkspace.
func (miw *morseImportWorkspace) addUpokt(addr string, amount cosmosmath.Int) error {
	importAccountIdx, hasAccountAddr := miw.addressToIdx[addr]
	if !hasAccountAddr {
		return ErrMorseStateTransform.Wrapf("account %q not found", addr)
	}

	account := miw.accountState.Accounts[importAccountIdx]
	if len(account.Coins) != 1 {
		return ErrMorseStateTransform.Wrapf(
			"account %q has %d token denominations, expected upokt only: %s",
			addr, len(account.Coins), account.Coins,
		)
	}

	upoktCoins := account.Coins[0]
	if upoktCoins.Denom != volatile.DenomuPOKT {
		return fmt.Errorf(
			"account %q has %s token denomination, expected upokt only: %s",
			addr, upoktCoins.Denom, account.Coins,
		)
	}

	account.Coins[0].Amount = account.Coins[0].Amount.Add(amount)
	return nil
}
