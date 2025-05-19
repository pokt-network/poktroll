package migration

import (
	"fmt"
	"slices"
	"testing"

	cometcrypto "github.com/cometbft/cometbft/crypto/ed25519"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/app/volatile"
	events "github.com/pokt-network/poktroll/testutil/events"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testmigration"
	"github.com/pokt-network/poktroll/x/migration/recovery"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

// Generate a random address that is not in the account state
var unclaimableAddress = cometcrypto.GenPrivKey().PubKey().Address().String()

// actorTypeToRecoverableAddress holds collections of different types of Morse addresses
// used in recovery testing. Each field contains a list of addresses categorized by their
// type and validity status to test different recovery scenarios.
type actorTypeToRecoverableAddress struct {
	MorseEOA                 []string
	MorseModule              []string
	MorseInvalidTooLong      []string
	MorseInvalidTooShort     []string
	MorseNonHex              []string
	MorseApplication         []string
	MorseOrphanedApplication []string
	MorseValidator           []string
	MorseOrphanedValidator   []string
}

func (s *MigrationModuleTestSuite) TestRecoverMorseAccount_AllowListSuccess() {
	t := s.T()

	_, accountState, actorTypeToRecoverableAddress, err := initMigrationFixtures(t)
	require.NoError(t, err)

	shannonDestAddr := sample.AccAddress()
	invalidShannonDestAddr := "invalid_shannon_dest_address"

	tests := []struct {
		name               string
		shannonDestAddress string
		morseSrcAddress    string
		expectedError      error
	}{
		{
			name:               "recover morse application account",
			shannonDestAddress: shannonDestAddr,
			morseSrcAddress:    actorTypeToRecoverableAddress.MorseApplication[0],
		},
		{
			name:               "recover morse validator account",
			shannonDestAddress: shannonDestAddr,
			morseSrcAddress:    actorTypeToRecoverableAddress.MorseValidator[0],
		},
		{
			name:               "recover morse EOA account",
			shannonDestAddress: shannonDestAddr,
			morseSrcAddress:    actorTypeToRecoverableAddress.MorseEOA[0],
		},
		{
			name:               "recover morse module account",
			shannonDestAddress: shannonDestAddr,
			morseSrcAddress:    actorTypeToRecoverableAddress.MorseModule[0],
		},
		{
			name:               "recover morse orphaned application account",
			shannonDestAddress: shannonDestAddr,
			morseSrcAddress:    actorTypeToRecoverableAddress.MorseOrphanedApplication[0],
		},
		{
			name:               "recover morse orphaned validator account",
			shannonDestAddress: shannonDestAddr,
			morseSrcAddress:    actorTypeToRecoverableAddress.MorseOrphanedValidator[0],
		},
		{
			name:               "recover morse invalid address too long",
			shannonDestAddress: shannonDestAddr,
			morseSrcAddress:    actorTypeToRecoverableAddress.MorseInvalidTooLong[0],
		},
		{
			name:               "recover morse invalid address too short",
			shannonDestAddress: shannonDestAddr,
			morseSrcAddress:    actorTypeToRecoverableAddress.MorseInvalidTooShort[0],
		},
		{
			name:               "recover morse invalid non-hex address",
			shannonDestAddress: shannonDestAddr,
			morseSrcAddress:    actorTypeToRecoverableAddress.MorseNonHex[0],
		},
		{
			name:               "recover morse invalid address not in allowlist",
			shannonDestAddress: shannonDestAddr,
			morseSrcAddress:    accountState.Accounts[1].MorseSrcAddress, // This address is not in the allowlist
			expectedError: status.Error(
				codes.InvalidArgument,
				migrationtypes.ErrMorseRecoverableAccountClaim.Wrapf(
					// We use triple-escaped quotes (\\\" instead of \") because the error
					// message goes through multiple layers of string formatting and wrapping:
					// 1. First in the Wrapf call creating the error
					// 2. Then when the error is wrapped in the RPC response
					// 3. Finally when it's included in the transaction log
					"morse account \\\"%s\\\" is not recoverable",
					accountState.Accounts[1].MorseSrcAddress,
				).Error(),
			),
		},
		{
			name:               "recover morse invalid address not found",
			shannonDestAddress: shannonDestAddr,
			morseSrcAddress:    unclaimableAddress,
			expectedError: status.Error(
				codes.NotFound,
				migrationtypes.ErrMorseRecoverableAccountClaim.Wrapf(
					"no morse recoverable account exists with address \\\"%s\\\"",
					unclaimableAddress,
				).Error(),
			),
		},
		{
			name:               "recover morse account with invalid shannon destination address",
			shannonDestAddress: invalidShannonDestAddr,
			morseSrcAddress:    actorTypeToRecoverableAddress.MorseInvalidTooLong[1],
			expectedError:      fmt.Errorf("invalid shannon destination address"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s.SetMorseAccountState(t, accountState)
			s.ImportMorseClaimableAccounts(t)

			morseClaimMsg := &migrationtypes.MsgRecoverMorseAccount{
				Authority:          authtypes.NewModuleAddress(govtypes.ModuleName).String(),
				ShannonDestAddress: test.shannonDestAddress,
				MorseSrcAddress:    test.morseSrcAddress,
			}

			resAny, err := s.GetApp().RunMsg(t, morseClaimMsg)
			if test.expectedError != nil {
				require.Error(t, err)
				require.ErrorContains(t, err, test.expectedError.Error())
				return
			}
			require.NoError(t, err)

			msgRecoveryRes, ok := resAny.(*migrationtypes.MsgRecoverMorseAccountResponse)
			require.True(t, ok)

			currentHeight := s.SdkCtx().BlockHeight() - 1
			sessionEndHeight := s.GetSessionEndHeight(t, currentHeight)

			claimedAccountIdx := slices.IndexFunc(accountState.Accounts, func(account *migrationtypes.MorseClaimableAccount) bool {
				return account.MorseSrcAddress == test.morseSrcAddress
			})
			claimedAccount := accountState.Accounts[claimedAccountIdx]
			claimedAccountBalance := claimedAccount.UnstakedBalance.
				Add(claimedAccount.ApplicationStake).
				Add(claimedAccount.SupplierStake)

			expectedRecoveryRes := &migrationtypes.MsgRecoverMorseAccountResponse{
				ShannonDestAddress: shannonDestAddr,
				MorseSrcAddress:    test.morseSrcAddress,
				RecoveredBalance:   claimedAccountBalance,
				SessionEndHeight:   sessionEndHeight,
			}
			require.Equal(t, msgRecoveryRes, expectedRecoveryRes)

			allEvents := s.GetApp().GetSdkCtx().EventManager().Events()
			eventMorseAccountRecovered := events.FilterEvents[*migrationtypes.EventMorseAccountRecovered](t, allEvents)
			require.Len(t, eventMorseAccountRecovered, 1)

			expectedEventMorseAccountRecovered := &migrationtypes.EventMorseAccountRecovered{
				SessionEndHeight:   sessionEndHeight,
				RecoveredBalance:   claimedAccountBalance,
				ShannonDestAddress: shannonDestAddr,
				MorseSrcAddress:    test.morseSrcAddress,
			}
			require.Equal(t, expectedEventMorseAccountRecovered, eventMorseAccountRecovered[0])
		})
	}
}

// initMigrationFixtures prepares test fixtures for Morse account recovery testing.
// It creates a variety of account types (valid, invalid, orphaned) and selectively adds
// some to a recovery allowlist to test different scenarios of the account recovery process.
//
// This function configures accounts with specific balances and stakes to create
// predictable test scenarios for the migration recovery process.
func initMigrationFixtures(t *testing.T) (
	*migrationtypes.MorseStateExport,
	*migrationtypes.MorseAccountState,
	*actorTypeToRecoverableAddress,
	error,
) {
	// Step 1: Configure different types of accounts for testing

	// Configure valid accounts (regular accounts, applications, validators, module accounts)
	validAccountsConfig := testmigration.ValidAccountsConfig{
		NumAccounts:       3, // Standard EOA accounts
		NumApplications:   3, // Application accounts with stake
		NumValidators:     3, // Validator accounts with stake
		NumModuleAccounts: 3, // System module accounts
	}

	// Configure accounts with invalid addresses to test error handling
	invalidAccountsConfig := testmigration.InvalidAccountsConfig{
		NumAddressTooShort: 2, // Addresses with fewer than expected bytes
		NumAddressTooLong:  2, // Addresses with more than expected bytes
		NumNonHexAddress:   2, // Addresses with non-hexadecimal characters
	}

	// Configure orphaned actors (applications and validators without corresponding accounts)
	// Used for testing recovery of unclaimed stakes
	orphanedActors := testmigration.OrphanedActorsConfig{
		NumApplications: 3, // Orphaned application actors
		NumValidators:   3, // Orphaned validator actors
	}

	// Step 2: Initialize the recovery allowlist and address categorization structure

	// The recovery allowlist will contain addresses that are eligible for recovery
	// For testing purposes, we add an address that would be contained in the allowlist
	// but is not actually present in the account state to simulate a recovery scenario.
	recoveryAllowlist := []string{unclaimableAddress}

	// This structure categorizes addresses by their type for focused testing
	// Each field will track addresses of a specific type that are added to the recovery allowlist
	actorTypeToRecoverableAddress := &actorTypeToRecoverableAddress{
		MorseEOA:                 []string{}, // Regular externally owned accounts
		MorseModule:              []string{}, // Module accounts (system accounts)
		MorseInvalidTooLong:      []string{}, // Accounts with addresses that are too long
		MorseInvalidTooShort:     []string{}, // Accounts with addresses that are too short
		MorseNonHex:              []string{}, // Accounts with non-hexadecimal addresses
		MorseApplication:         []string{}, // Application accounts with stakes
		MorseOrphanedApplication: []string{}, // Application accounts without corresponding base accounts
		MorseValidator:           []string{}, // Validator accounts with stakes
		MorseOrphanedValidator:   []string{}, // Validator accounts without corresponding base accounts
	}

	// Step 3: Define callback functions to customize account properties

	// This function determines the unstaked balance for each account type
	// It also selects specific accounts to add to the recovery allowlist for testing
	unstakedAccountBalancesFn := func(
		allActorsIndex, // Index across all actor types
		actorsIndex uint64, // Index within the specific actor type
		actorType testmigration.MorseUnstakedActorType, // The type of actor being processed
		morseAccount *migrationtypes.MorseAccount, // The account being processed
	) *cosmostypes.Coin {
		// All unstaked accounts get an initial balance of 1000 uPOKT
		coin := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 1000)

		// Based on actor type, selectively add the first account of each type to the recovery allowlist
		switch actorType {
		// If the actor type is the first module account, append it to the recovery
		// allowlist to exercise the recovery logic for these cases.
		case testmigration.MorseModule:
			if actorsIndex == 0 {
				actorTypeToRecoverableAddress.MorseModule = append(
					actorTypeToRecoverableAddress.MorseModule,
					morseAccount.Address.String(),
				)
				recoveryAllowlist = append(recoveryAllowlist, morseAccount.Address.String())
			}
		// If the actor type is the first valid EOA account, append it to the recovery
		// allowlist to exercise the recovery logic for these cases.
		case testmigration.MorseEOA:
			if actorsIndex == 0 {
				actorTypeToRecoverableAddress.MorseEOA = append(
					actorTypeToRecoverableAddress.MorseEOA,
					morseAccount.Address.String(),
				)
				recoveryAllowlist = append(recoveryAllowlist, morseAccount.Address.String())
			}
		// Add any invalid morse accounts to the recovery allowlist.
		case testmigration.MorseInvalidTooLong:
			actorTypeToRecoverableAddress.MorseInvalidTooLong = append(
				actorTypeToRecoverableAddress.MorseInvalidTooLong,
				morseAccount.Address.String(),
			)
			recoveryAllowlist = append(recoveryAllowlist, morseAccount.Address.String())
		case testmigration.MorseInvalidTooShort:
			actorTypeToRecoverableAddress.MorseInvalidTooShort = append(
				actorTypeToRecoverableAddress.MorseInvalidTooShort,
				morseAccount.Address.String(),
			)
			recoveryAllowlist = append(recoveryAllowlist, morseAccount.Address.String())
		case testmigration.MorseNonHex:
			actorTypeToRecoverableAddress.MorseNonHex = append(
				actorTypeToRecoverableAddress.MorseNonHex,
				morseAccount.Address.String(),
			)
			recoveryAllowlist = append(recoveryAllowlist, morseAccount.Address.String())
		}

		return &coin
	}

	// This function determines the staked and unstaked balances for application accounts
	// It also adds the first application of each type to the recovery allowlist
	applicationStakesFn := func(
		index uint64, // Global index across all applications
		actorTypeIndex uint64, // Index within the specific application type
		actorType testmigration.MorseApplicationActorType, // The type of application (regular or orphaned)
		application *migrationtypes.MorseApplication, // The application account being processed
	) (staked, unstaked *cosmostypes.Coin) {
		// Applications have 2000 uPOKT staked and 1000 uPOKT unstaked balances
		stakedCoin := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 2000)
		unstakedCoin := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 1000)

		// If the application is the first one of its type (Application or OrphanedApplication),
		// append it to the recovery allowlist.
		if actorTypeIndex == 0 {
			switch actorType {
			case testmigration.MorseApplication:
				actorTypeToRecoverableAddress.MorseApplication = append(
					actorTypeToRecoverableAddress.MorseApplication,
					application.Address.String(),
				)
			case testmigration.MorseOrphanedApplication:
				actorTypeToRecoverableAddress.MorseOrphanedApplication = append(
					actorTypeToRecoverableAddress.MorseOrphanedApplication,
					application.Address.String(),
				)
			}
			// Append the application address to the recovery allowlist.
			recoveryAllowlist = append(recoveryAllowlist, application.Address.String())
		}

		return &stakedCoin, &unstakedCoin
	}

	// This function determines the staked and unstaked balances for validator accounts
	// It also adds the first validator of each type to the recovery allowlist
	validatorStakesFn := func(
		index uint64, // Global index across all validators
		actorTypeIndex uint64, // Index within the specific validator type
		actorType testmigration.MorseValidatorActorType, // The type of validator (regular or orphaned)
		validator *migrationtypes.MorseValidator, // The validator account being processed
	) (staked, unstaked *cosmostypes.Coin) {
		// Validators have 3000 uPOKT staked and 1000 uPOKT unstaked balances
		stakedCoin := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 3000)
		unstakedCoin := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 1000)

		// If the validator is the first one of its type (Validator or OrphanedValidator),
		// append it to the recovery allowlist.
		if actorTypeIndex == 0 {
			switch actorType {
			case testmigration.MorseValidator:
				actorTypeToRecoverableAddress.MorseValidator = append(
					actorTypeToRecoverableAddress.MorseValidator,
					validator.Address.String(),
				)
			case testmigration.MorseOrphanedValidator:
				actorTypeToRecoverableAddress.MorseOrphanedValidator = append(
					actorTypeToRecoverableAddress.MorseOrphanedValidator,
					validator.Address.String(),
				)
			}
			// Append the validator address to the recovery allowlist.
			recoveryAllowlist = append(recoveryAllowlist, validator.Address.String())
		}

		return &stakedCoin, &unstakedCoin
	}

	// This function generates unique names for module accounts based on their index
	moduleAccountNameFn := func(index, actorTypeIndex uint64) string {
		return fmt.Sprintf("module-account-%d", actorTypeIndex)
	}

	// Step 4: Create the Morse fixtures with the defined configurations and callback functions
	fixtures, err := testmigration.NewMorseFixtures(
		// Configure the types and numbers of accounts to create
		testmigration.WithValidAccounts(validAccountsConfig),
		testmigration.WithInvalidAccounts(invalidAccountsConfig),
		testmigration.WithOrphanedActors(orphanedActors),

		// Configure the balance and stake amounts for each account type
		testmigration.WithUnstakedAccountBalancesFn(unstakedAccountBalancesFn),
		testmigration.WithApplicationStakesFn(applicationStakesFn),
		testmigration.WithValidatorStakesFn(validatorStakesFn),

		// Configure module account naming
		testmigration.WithModuleAccountNameFn(moduleAccountNameFn),
	)
	require.NoError(t, err)

	// Step 5: Extract the state export and account state from the fixtures
	stateExport := fixtures.GetMorseStateExport()
	accountState := fixtures.GetMorseAccountState()

	// Step 6: Register the recovery allowlist with the recovery system
	// This makes the selected addresses eligible for recovery in tests
	recovery.SetRecoveryAllowlist(recoveryAllowlist)

	return stateExport, accountState, actorTypeToRecoverableAddress, err
}
