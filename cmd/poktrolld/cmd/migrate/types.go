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
		accountIdxByAddress: make(map[string]uint64),
		accountState: &migrationtypes.MorseAccountState{
			Accounts: make([]*migrationtypes.MorseClaimableAccount, 0),
		},
		accumulatedTotalBalance:       cosmosmath.ZeroInt(),
		accumulatedTotalAppStake:      cosmosmath.ZeroInt(),
		accumulatedTotalSupplierStake: cosmosmath.ZeroInt(),
	}
}

// morseImportWorkspace is a helper struct that is used to consolidate the Morse account balance,
// application stake, and supplier stake for each account as an entry in the resulting MorseAccountState.
type morseImportWorkspace struct {
	// accountIdxByAddress is a map from the Shannon bech32 address to the index of the
	// corresponding MorseAccount in the accounts slice.
	accountIdxByAddress map[string]uint64
	// accountState is the final MorseAccountState that will be imported into Shannon.
	// It includes a slice of MorseAccount objects which are populated, by transforming
	// the input MorseStateExport into the output MorseAccountState.
	accountState *migrationtypes.MorseAccountState

	// accumulatedTotalBalance is the most recently accumulated balances of all Morse
	// accounts which have been processed.
	accumulatedTotalBalance cosmosmath.Int
	// accumulatedTotalAppStake is the most recently accumulated application stakes of
	// all Morse accounts which have been processed.
	accumulatedTotalAppStake cosmosmath.Int
	// accumulatedTotalSupplierStake is the most recently accumulated supplier stakes of
	// all Morse accounts which have been processed.
	accumulatedTotalSupplierStake cosmosmath.Int
	// numApplications is the number of applications that have been processed.
	numApplications uint64
	// numSuppliers is the number of suppliers that have been processed.
	numSuppliers uint64
}

// nextIdx returns the next index to be used when appending a new account to the accounts slice.
func (miw *morseImportWorkspace) nextIdx() int64 {
	return int64(len(miw.accountState.GetAccounts()))
}

// getAccount returns the MorseAccount for the given address and its index,
// if present, in the accountState accounts slice.
// If the given address is not present, it returns nil, -1.
func (miw *morseImportWorkspace) getAccount(addr string) (*migrationtypes.MorseClaimableAccount, int64) {
	accountIdx, ok := miw.accountIdxByAddress[addr]
	if !ok {
		return nil, -1
	}

	account := miw.accountState.GetAccounts()[accountIdx]
	return account, int64(accountIdx)
}

// hasAccount returns true if the given address is present in the accounts slice.
func (miw *morseImportWorkspace) hasAccount(addr string) bool {
	_, accountIdx := miw.getAccount(addr)
	return accountIdx != -1
}

// getNumAccounts returns the number of accounts in the accountState accounts map.
func (miw *morseImportWorkspace) getNumAccounts() uint64 {
	return uint64(len(miw.accountState.GetAccounts()))
}

// infoLogComplete prints info level logs indicating the completion of the import.
func (miw *morseImportWorkspace) infoLogComplete() error {
	accountStateHash, err := miw.accountState.GetHash()
	if err != nil {
		return err
	}

	logger.Info().
		Uint64("num_accounts", miw.getNumAccounts()).
		Uint64("num_applications", miw.numApplications).
		Uint64("num_suppliers", miw.numSuppliers).
		Str("total_balance", miw.accumulatedTotalBalance.String()).
		Str("total_app_stake", miw.accumulatedTotalAppStake.String()).
		Str("total_supplier_stake", miw.accumulatedTotalSupplierStake.String()).
		Str("grand_total", miw.accumulatedTotalsSum().String()).
		Str("morse_account_state_hash", fmt.Sprintf("%x", accountStateHash)).
		Msg("processing accounts complete")
	return nil
}

// accumulatedTotalsSum returns the sum of the accumulatedTotalBalance,
// accumulatedTotalAppStake, and accumulatedTotalSupplierStake.
func (miw *morseImportWorkspace) accumulatedTotalsSum() cosmosmath.Int {
	return miw.accumulatedTotalBalance.
		Add(miw.accumulatedTotalAppStake).
		Add(miw.accumulatedTotalSupplierStake)
}

// addAccount adds the account with the given address to the accounts slice and
// its corresponding address is in the accountIdxByAddress map.
// If the address is already present, an error is returned.
func (miw *morseImportWorkspace) addAccount(
	addr string,
	exportAccount *migrationtypes.MorseAuthAccount,
) (accountIdx int64, balance cosmostypes.Coin, err error) {
	// Initialize balance to zero
	balance = cosmostypes.NewCoin(volatile.DenomuPOKT, cosmosmath.ZeroInt())

	if _, accountIdx = miw.getAccount(addr); accountIdx != -1 {
		return 0, cosmostypes.Coin{}, ErrMorseStateTransform.Wrapf(
			"unexpected workspace state: account already exists (%s)", addr,
		)
	}

	accountIdx = miw.nextIdx()
	importAccount := &migrationtypes.MorseClaimableAccount{
		Address:     exportAccount.Value.Address,
		PublicKey:   exportAccount.Value.PubKey.Value,
		TotalTokens: cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 0),
	}
	miw.accountState.Accounts = append(miw.accountState.Accounts, importAccount)
	miw.accountIdxByAddress[addr] = uint64(accountIdx)

	return accountIdx, balance, nil
}

// addUpokt adds the given amount to the corresponding account balances in the morseWorkspace.
func (miw *morseImportWorkspace) addUpokt(addr string, amount cosmosmath.Int) error {
	account, accountIdx := miw.getAccount(addr)
	if accountIdx == -1 {
		return ErrMorseStateTransform.Wrapf("account %q not found", addr)
	}

	if account.TotalTokens.Denom != volatile.DenomuPOKT {
		return fmt.Errorf(
			"account %q has %s token denomination, expected upokt only: %s",
			addr, account.TotalTokens.Denom, account.TotalTokens,
		)
	}

	account.TotalTokens.Amount = account.TotalTokens.Amount.Add(amount)
	return nil
}
