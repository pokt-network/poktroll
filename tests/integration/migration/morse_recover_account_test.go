//go:build test

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

	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/testutil/events"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testmigration"
	"github.com/pokt-network/poktroll/x/migration/recovery"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

// Generate a random address that is not in the account state
var unclaimableAddress = cometcrypto.GenPrivKey().PubKey().Address().String()

func (s *MigrationModuleTestSuite) TestRecoverMorseAccount_AllowListSuccess() {
	t := s.T()

	fixtures, err := initMigrationFixtures(t)
	require.NoError(t, err)

	// Valid shannon destination address to be reused in all tests
	shannonDestAddr := sample.AccAddress()
	invalidShannonDestAddr := "invalid_shannon_dest_address"

	// Get the complete state of all Morse accounts for testing
	accountState := fixtures.GetMorseAccountState()

	// Get standard and orphaned application accounts with stake that can be recovered
	applicationFixtures := fixtures.GetApplicationFixtures(testmigration.MorseApplication)
	orphanedApplicationFixtures := fixtures.GetApplicationFixtures(testmigration.MorseOrphanedApplication)

	// Get standard and orphaned validator accounts with stake that can be recovered
	validatorFixtures := fixtures.GetValidatorFixtures(testmigration.MorseValidator)
	orphanedValidatorFixtures := fixtures.GetValidatorFixtures(testmigration.MorseOrphanedValidator)

	// Get the unstaked accounts grouped by their type
	eoaFixtures := fixtures.GetUnstakedActorFixtures(testmigration.MorseEOA)
	moduleFixtures := fixtures.GetUnstakedActorFixtures(testmigration.MorseModule)
	addressTooLongAccountFixtures := fixtures.GetUnstakedActorFixtures(testmigration.MorseInvalidTooLong)
	addressTooShortAccountFixtures := fixtures.GetUnstakedActorFixtures(testmigration.MorseInvalidTooShort)
	nonHexAccountFixtures := fixtures.GetUnstakedActorFixtures(testmigration.MorseNonHex)

	tests := []struct {
		name               string
		shannonDestAddress string
		morseSrcAddress    string
		expectedError      error
	}{
		{
			name:               "recover morse application account",
			shannonDestAddress: shannonDestAddr,
			morseSrcAddress:    applicationFixtures[0].GetClaimableAccount().MorseSrcAddress,
		},
		{
			name:               "recover morse validator account",
			shannonDestAddress: shannonDestAddr,
			morseSrcAddress:    validatorFixtures[0].GetClaimableAccount().MorseSrcAddress,
		},
		{
			name:               "recover morse EOA account",
			shannonDestAddress: shannonDestAddr,
			morseSrcAddress:    eoaFixtures[0].GetClaimableAccount().MorseSrcAddress,
		},
		{
			name:               "recover morse module account",
			shannonDestAddress: shannonDestAddr,
			morseSrcAddress:    moduleFixtures[0].GetClaimableAccount().MorseSrcAddress,
		},
		{
			name:               "recover morse orphaned application account",
			shannonDestAddress: shannonDestAddr,
			morseSrcAddress:    orphanedApplicationFixtures[0].GetClaimableAccount().MorseSrcAddress,
		},
		{
			name:               "recover morse orphaned validator account",
			shannonDestAddress: shannonDestAddr,
			morseSrcAddress:    orphanedValidatorFixtures[0].GetClaimableAccount().MorseSrcAddress,
		},
		{
			name:               "recover morse invalid address too long",
			shannonDestAddress: shannonDestAddr,
			morseSrcAddress:    addressTooLongAccountFixtures[0].GetClaimableAccount().MorseSrcAddress,
		},
		{
			name:               "recover morse invalid address too short",
			shannonDestAddress: shannonDestAddr,
			morseSrcAddress:    addressTooShortAccountFixtures[0].GetClaimableAccount().MorseSrcAddress,
		},
		{
			name:               "recover morse invalid non-hex address",
			shannonDestAddress: shannonDestAddr,
			morseSrcAddress:    nonHexAccountFixtures[0].GetClaimableAccount().MorseSrcAddress,
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
			morseSrcAddress:    eoaFixtures[0].GetClaimableAccount().MorseSrcAddress,
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
			claimedAccountBalance := claimedAccount.TotalTokens()

			expectedRecoveryRes := &migrationtypes.MsgRecoverMorseAccountResponse{}
			require.Equal(t, msgRecoveryRes, expectedRecoveryRes)

			allEvents := s.GetApp().GetSdkCtx().EventManager().Events()
			eventMorseAccountRecovered := events.FilterEvents[*migrationtypes.EventMorseAccountRecovered](t, allEvents)
			require.Len(t, eventMorseAccountRecovered, 1)

			expectedEventMorseAccountRecovered := &migrationtypes.EventMorseAccountRecovered{
				SessionEndHeight:   sessionEndHeight,
				RecoveredBalance:   claimedAccountBalance.String(),
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
func initMigrationFixtures(t *testing.T) (*testmigration.MorseMigrationFixtures, error) {
	// Step 1: Configure different types of accounts for testing

	// Configure valid accounts (regular accounts, applications, validators, module accounts).
	validAccountsConfig := testmigration.ValidAccountsConfig{
		NumAccountsValid:     3, // Standard EOA accounts
		NumApplicationsValid: 3, // Application accounts with stake
		NumValidatorsValid:   3, // Validator accounts with stake
		NumModuleAccounts:    3, // System module accounts
	}

	// Configure accounts with invalid addresses to test error handling.
	invalidAccountsConfig := testmigration.InvalidAccountsConfig{
		NumAddressTooShort: 2, // Addresses with fewer than expected bytes
		NumAddressTooLong:  2, // Addresses with more than expected bytes
		NumNonHexAddress:   2, // Addresses with non-hexadecimal characters
	}

	// Configure orphaned actors (applications and validators without corresponding accounts).
	// Used for testing recovery of unclaimed stakes.
	orphanedActors := testmigration.OrphanedActorsConfig{
		NumApplicationsOrphaned: 3, // Orphaned application actors
		NumValidatorsOrphaned:   3, // Orphaned validator actors
	}

	// Step 2: Initialize the recovery allowlist and address categorization structure

	// The recovery allowlist will contain addresses that are eligible for recovery.
	// For testing purposes, we add an address that would be contained in the allowlist
	// but is not actually present in the account state to simulate a recovery scenario.
	recoveryAllowlist := []string{unclaimableAddress}

	// Step 3: Define callback functions to customize account properties

	// This function determines the unstaked balance for each account type
	// It also selects specific accounts to add to the recovery allowlist for testing
	unstakedAccountBalancesFn := func(
		allAccountsIndex, // Index across all accounts types
		actorIndex uint64, // Index within the specific actor type
		actorType testmigration.MorseUnstakedActorType, // The type of actor being processed
		morseAccount *migrationtypes.MorseAccount, // The account being processed
	) *cosmostypes.Coin {
		// All unstaked accounts get an initial balance of 1000 uPOKT
		coin := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 1000)

		// Based on actor type, selectively add the first account of each type to the recovery allowlist
		switch actorType {
		// If the actor type is the first module account, append it to the recovery
		// allowlist to exercise the recovery logic for these cases.
		case testmigration.MorseModule:
			if actorIndex == 0 {
				recoveryAllowlist = append(recoveryAllowlist, morseAccount.Address.String())
			}
		// If the actor type is the first valid EOA account, append it to the recovery
		// allowlist to exercise the recovery logic for these cases.
		case testmigration.MorseEOA:
			if actorIndex == 0 {
				recoveryAllowlist = append(recoveryAllowlist, morseAccount.Address.String())
			}
		// Add any invalid morse accounts to the recovery allowlist.
		case testmigration.MorseInvalidTooLong:
			recoveryAllowlist = append(recoveryAllowlist, morseAccount.Address.String())
		case testmigration.MorseInvalidTooShort:
			recoveryAllowlist = append(recoveryAllowlist, morseAccount.Address.String())
		case testmigration.MorseNonHex:
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
		stakedCoin := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 2000)
		unstakedCoin := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 1000)

		// If the application is the first one of its type (Application or OrphanedApplication),
		// append it to the recovery allowlist.
		if actorTypeIndex == 0 {
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
		stakedCoin := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 3000)
		unstakedCoin := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 1000)

		// If the validator is the first one of its type (Validator or OrphanedValidator),
		// append it to the recovery allowlist.
		if actorTypeIndex == 0 {
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

	// Step 5: Register the recovery allowlist with the recovery system
	// This makes the selected addresses eligible for recovery in tests
	recovery.SetRecoveryAllowlist(recoveryAllowlist)

	return fixtures, err
}
