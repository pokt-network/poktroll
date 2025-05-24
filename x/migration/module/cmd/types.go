package cmd

import (
	"fmt"

	cosmosmath "cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/cmd/logger"
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
	// accountIdxByAddress is a map from the hex-encoded Morse address to the index
	// of the corresponding MorseAccount in the accounts slice.
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
func (miw *morseImportWorkspace) getAccount(morseAddress string) (*migrationtypes.MorseClaimableAccount, error) {
	accountIdx, ok := miw.accountIdxByAddress[morseAddress]
	if !ok {
		return nil, ErrMorseStateTransform.Wrapf("account %q not found", morseAddress)
	}

	account := miw.accountState.GetAccounts()[accountIdx]
	return account, nil
}

// hasAccount returns true if the given address is present in the accounts slice.
func (miw *morseImportWorkspace) hasAccount(morseAddress string) bool {
	_, err := miw.getAccount(morseAddress)
	return err == nil
}

// getNumAccounts returns the number of Morse accounts in the accountState accounts map.
func (miw *morseImportWorkspace) getNumAccounts() uint64 {
	return uint64(len(miw.accountState.GetAccounts()))
}

// infoLogComplete prints info level logs indicating the completion of the import.
func (miw *morseImportWorkspace) infoLogComplete() error {
	accountStateHash, err := miw.accountState.GetHash()
	if err != nil {
		return err
	}

	logger.Logger.Info().
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

// addAccount adds the Morse account with the given Morse address to the accounts
// slice and its corresponding address is in the accountIdxByAddress map.
// - If the Morse account is a module account, the module account name is used as the MorseClaimableAccount address
// - If the address is already present, an error is returned
func (miw *morseImportWorkspace) addAccount(addr string) error {
	if _, err := miw.getAccount(addr); err == nil {
		logger.Logger.Warn().
			Str("address", addr).
			Msg("account already exists, stakes and balances will be summed")
		return nil
	}

	accountIdx := miw.nextIdx()
	importAccount := &migrationtypes.MorseClaimableAccount{
		MorseSrcAddress:  addr,
		UnstakedBalance:  cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 0),
		SupplierStake:    cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 0),
		ApplicationStake: cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 0),
	}
	miw.accountState.Accounts = append(miw.accountState.Accounts, importAccount)
	miw.accountIdxByAddress[addr] = uint64(accountIdx)

	return nil
}

// addUnstakedBalance adds the given amount to the corresponding Morse account balances in the morseWorkspace.
func (miw *morseImportWorkspace) addUnstakedBalance(addr string, amount cosmosmath.Int) error {
	account, err := miw.getAccount(addr)
	if err != nil {
		return err
	}

	account.UnstakedBalance.Amount = account.UnstakedBalance.Amount.Add(amount)
	return nil
}

// addSupplierStake does two things:
// - Adds the given amount to the corresponding Morse account balances in the morseWorkspace
// - Sets the MorseOutputAddress if the given outputAddr is not nil
func (miw *morseImportWorkspace) addSupplierStake(morseSupplier *migrationtypes.MorseValidator) error {
	// Retrieve the Morse supplier (aka Service/Node) account
	morseClaimableAccount, err := miw.getAccount(morseSupplier.Address.String())
	if err != nil {
		return err
	}

	// Update the supplier stake amount
	supplierStakeAmtUpokt, ok := cosmosmath.NewIntFromString(morseSupplier.StakedTokens)
	if !ok {
		return ErrMorseExportState.Wrapf("failed to parse supplier stake amount %q", morseSupplier.StakedTokens)
	}
	morseClaimableAccount.SupplierStake.Amount = morseClaimableAccount.SupplierStake.Amount.
		Add(supplierStakeAmtUpokt)

	// Custodial address (i.e. output, a.k.a. owner) is optional.
	if morseSupplier.OutputAddress != nil {
		morseClaimableAccount.MorseOutputAddress = morseSupplier.OutputAddress.String()
	}

	// If the supplier is unbonding, transfer the unstaking completion time.
	if !morseSupplier.UnstakingTime.IsZero() {
		morseClaimableAccount.UnstakingTime = morseSupplier.UnstakingTime
	}

	miw.accumulatedTotalSupplierStake = miw.accumulatedTotalSupplierStake.Add(supplierStakeAmtUpokt)
	miw.numSuppliers++

	return nil
}

// addAppStake adds the given amount to the corresponding Morse account balances in the morseWorkspace.
func (miw *morseImportWorkspace) addAppStake(morseApplication *migrationtypes.MorseApplication) error {
	appStakeAmtUpokt, ok := cosmosmath.NewIntFromString(morseApplication.StakedTokens)
	if !ok {
		return ErrMorseExportState.Wrapf("failed to parse application stake amount %q", morseApplication.StakedTokens)
	}

	// Retrieve the Morse application (aka Validator) account
	morseClaimableAccount, err := miw.getAccount(morseApplication.Address.String())
	if err != nil {
		return err
	}

	// If the application is unbonding, transfer the unstaking completion time.
	if !morseApplication.UnstakingTime.IsZero() {
		morseClaimableAccount.UnstakingTime = morseApplication.UnstakingTime
	}

	morseClaimableAccount.ApplicationStake.Amount = morseClaimableAccount.ApplicationStake.Amount.Add(appStakeAmtUpokt)

	miw.accumulatedTotalAppStake = miw.accumulatedTotalAppStake.Add(appStakeAmtUpokt)
	miw.numApplications++

	return nil
}
