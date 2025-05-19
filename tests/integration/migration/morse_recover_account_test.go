package migration

import (
	"fmt"
	"slices"
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testmigration"
	"github.com/pokt-network/poktroll/x/migration/recovery"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

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

func (s *MigrationModuleTestSuite) TestRecoverMorseAccount_AllowListApp() {
	t := s.T()
	_, accountState, actorTypeToRecoverableAddress, err := initMigrationFixtures(t)
	require.NoError(t, err)

	s.SetMorseAccountState(t, accountState)
	s.ImportMorseClaimableAccounts(t)

	shannonDestAddr := sample.AccAddress()

	morseApplicationAddress := actorTypeToRecoverableAddress.MorseApplication[0]
	morseClaimMsg := &migrationtypes.MsgRecoverMorseAccount{
		Authority:          authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		ShannonDestAddress: shannonDestAddr,
		MorseSrcAddress:    morseApplicationAddress,
	}

	resAny, err := s.GetApp().RunMsg(t, morseClaimMsg)
	require.NoError(t, err)

	msgRecoveryRes, ok := resAny.(*migrationtypes.MsgRecoverMorseAccountResponse)
	require.True(t, ok)

	currentHeight := s.SdkCtx().BlockHeight() - 1
	sessionEndHeight := s.GetSessionEndHeight(t, currentHeight)

	claimedAccountIdx := slices.IndexFunc(accountState.Accounts, func(account *migrationtypes.MorseClaimableAccount) bool {
		return account.MorseSrcAddress == morseApplicationAddress
	})
	claimedAccount := accountState.Accounts[claimedAccountIdx]
	claimedAccountBalance := claimedAccount.UnstakedBalance.Add(claimedAccount.ApplicationStake)

	expectedRecoveryRes := &migrationtypes.MsgRecoverMorseAccountResponse{
		ShannonDestAddress: shannonDestAddr,
		MorseSrcAddress:    morseApplicationAddress,
		RecoveredBalance:   claimedAccountBalance,
		SessionEndHeight:   sessionEndHeight,
	}

	require.Equal(t, msgRecoveryRes, expectedRecoveryRes)
}

func TestRecoverMorseAccount_AllowListSupplier(t *testing.T) {
}

func TestRecoverMorseAccount_AccountAddressTooShort(t *testing.T) {
}

func TestRecoverMorseAccount_AccountAddressTooLong(t *testing.T) {
}

func TestRecoverMorseAccount_AccountAddressNonHex(t *testing.T) {
}

func TestRecoverMorseAccount_ModuleAccount(t *testing.T) {
}

// initMigrationFixtures initializes the migration fixtures for testing.
func initMigrationFixtures(t *testing.T) (
	*migrationtypes.MorseStateExport,
	*migrationtypes.MorseAccountState,
	*actorTypeToRecoverableAddress,
	error,
) {
	validAccountsConfig := testmigration.ValidAccountsConfig{
		NumAccounts:       3,
		NumApplications:   3,
		NumValidators:     3,
		NumModuleAccounts: 3,
	}

	invalidAccountsConfig := testmigration.InvalidAccountsConfig{
		NumAddressTooShort: 2,
		NumAddressTooLong:  2,
		NumNonHexAddress:   2,
	}

	orphanedActors := testmigration.OrphanedActorsConfig{
		NumApplications: 3,
		NumValidators:   3,
	}

	recoveryAllowlist := []string{}
	actorTypeToRecoverableAddress := &actorTypeToRecoverableAddress{
		MorseEOA:                 []string{},
		MorseModule:              []string{},
		MorseInvalidTooLong:      []string{},
		MorseInvalidTooShort:     []string{},
		MorseNonHex:              []string{},
		MorseApplication:         []string{},
		MorseOrphanedApplication: []string{},
		MorseValidator:           []string{},
		MorseOrphanedValidator:   []string{},
	}

	unstakedAccountBalancesFn := func(
		allActorsIndex,
		actorsIndex uint64,
		actorType testmigration.MorseUnstakedActorType,
		morseAccount *migrationtypes.MorseAccount,
	) *cosmostypes.Coin {
		coin := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 1000)

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

	applicationStakesFn := func(
		index, actorTypeIndex uint64,
		actorType testmigration.MorseApplicationActorType,
		application *migrationtypes.MorseApplication,
	) (staked, unstaked *cosmostypes.Coin) {
		stakedCoin := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 2000)
		unstakedCoin := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 1000)

		// If the validator is the first one Application or OrphanedApplication,
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

	validatorStakesFn := func(
		index, actorTypeIndex uint64,
		actorType testmigration.MorseValidatorActorType,
		validator *migrationtypes.MorseValidator,
	) (staked, unstaked *cosmostypes.Coin) {
		stakedCoin := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 3000)
		unstakedCoin := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 1000)

		// If the validator is the first one Validator or OrphanedValidator,
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

	moduleAccountNameFn := func(index, actorTypeIndex uint64) string {
		return fmt.Sprintf("module-account-%d", actorTypeIndex)
	}

	fixtures, err := testmigration.NewMorseFixtures(
		testmigration.WithValidAccounts(validAccountsConfig),
		testmigration.WithInvalidAccounts(invalidAccountsConfig),
		testmigration.WithOrphanedActors(orphanedActors),
		testmigration.WithUnstakedAccountBalancesFn(unstakedAccountBalancesFn),
		testmigration.WithApplicationStakesFn(applicationStakesFn),
		testmigration.WithValidatorStakesFn(validatorStakesFn),
		testmigration.WithModuleAccountNameFn(moduleAccountNameFn),
	)
	require.NoError(t, err)

	stateExport := fixtures.GetMorseStateExport()
	accountState := fixtures.GetMorseAccountState()

	recovery.SetRecoveryAllowlist(recoveryAllowlist)

	return stateExport, accountState, actorTypeToRecoverableAddress, err
}
