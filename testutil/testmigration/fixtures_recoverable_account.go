package testmigration

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"time"

	"cosmossdk.io/math"
	"github.com/cometbft/cometbft/crypto"
	cometcrypto "github.com/cometbft/cometbft/crypto/ed25519"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/app/pocket"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

// MorseUnstakedActorType represents different types of unstaked Morse accounts
// that can be included in test fixtures.
type MorseUnstakedActorType int

const (
	// MorseEOA represents a standard externally owned account (EOA)
	MorseEOA = MorseUnstakedActorType(iota)
	// MorseInvalidTooShort represents an account with an address that is too short to be valid
	MorseInvalidTooShort
	// MorseInvalidTooLong represents an account with an address that is too long to be valid
	MorseInvalidTooLong
	// MorseNonHex represents an account with an address that contains non-hexadecimal characters
	MorseNonHex
	// MorseModule represents a module account
	MorseModule
	// MorseNonCustodialOwnerAccount represents a non-custodial validator's owner account
	MorseNonCustodialOwnerAccount
)

// MorseValidatorActorType represents different types of validator actors
// in Morse that can be included in test fixtures.
type MorseValidatorActorType int

const (
	// MorseValidator represents a standard validator with both staked and unstaked accounts
	MorseValidator = MorseValidatorActorType(iota)
	// MorseOrphanedValidator represents a validator without a corresponding unstaked account
	MorseOrphanedValidator
	// MorseUnbondingValidator represents a validator that has begun unbonding on Morse
	MorseUnbondingValidator
	// MorseUnbondedValidator represents a validator that has unbonded on Morse while waiting to be claimed
	MorseUnbondedValidator
	// MorseNonCustodialValidator represents a non-custodial validator with a separate owner account
	MorseNonCustodialValidator
)

// MorseApplicationActorType represents different types of application actors
// in Morse that can be included in test fixtures.
type MorseApplicationActorType int

const (
	// MorseApplication represents a standard application with both staked and unstaked accounts
	MorseApplication = MorseApplicationActorType(iota)
	// MorseOrphanedApplication represents an application without a corresponding unstaked account
	MorseOrphanedApplication
	// MorseUnbondingApplication represents an application that has begun unbonding on Morse
	MorseUnbondingApplication
	// MorseUnbondedApplication represents an application that has unbonded on Morse while waiting to be claimed
	MorseUnbondedApplication
)

// actorFixture represents a fixture for a Morse actor (account, validator, or application)
// that can be used in migration testing. It combines the actor itself with its claimable
// account representation and private key for signing purposes.
type actorFixture[T any] struct {
	actor            T                                     // The Morse actor (account, validator, or application)
	claimableAccount *migrationtypes.MorseClaimableAccount // The claimable account representation used during migration
	privKey          cometcrypto.PrivKey                   // The private key associated with this actor
}

// GetClaimableAccount returns the MorseClaimableAccount associated with this actor fixture,
// which contains information needed for the migration process.
func (af *actorFixture[T]) GetClaimableAccount() *migrationtypes.MorseClaimableAccount {
	return af.claimableAccount
}

// GetActor returns the underlying Morse actor (account, validator, or application)
// represented by this fixture.
func (af *actorFixture[T]) GetActor() T {
	return af.actor
}

// GetPrivateKey returns the private key associated with this actor fixture,
// which can be used for signing purposes.
func (af *actorFixture[T]) GetPrivateKey() cometcrypto.PrivKey {
	return af.privKey
}

// GetAddress returns the address which corresponds to the actor fixture.
func (af *actorFixture[T]) GetAddress() string {
	return af.privKey.PubKey().Address().String()
}

// actorTypeGroups organizes actor fixtures by their type, allowing for easy access
// to specific categories of actors in the test fixtures.
type actorTypeGroups struct {
	// Unstaked accounts grouped by type
	unstakedAccounts map[MorseUnstakedActorType][]*actorFixture[*migrationtypes.MorseAccount]
	// Validators grouped by type
	validators map[MorseValidatorActorType][]*actorFixture[*migrationtypes.MorseValidator]
	// Applications grouped by type
	applications map[MorseApplicationActorType][]*actorFixture[*migrationtypes.MorseApplication]
}

// MorseMigrationFixtures contains the state and configuration for generating Morse migration test fixtures.
// It maintains internal state for tracking and indexing Morse accounts, keys, and migration state.
type MorseMigrationFixtures struct {
	// config holds the configuration parameters for generating fixtures
	config *MorseFixturesConfig

	// morseStateExport contains the Morse blockchain state being exported
	morseStateExport *migrationtypes.MorseStateExport

	// morseAccountState contains the accounts that can be claimed in the migration process
	morseAccountState *migrationtypes.MorseAccountState

	// Tracks the current index for generating sequential keys/accounts
	currentIndex uint64

	// actorTypeGroups organizes actor fixtures by their type
	actorTypeGroups actorTypeGroups
}

// MorseFixturesConfig defines the configuration parameters for generating Morse migration test fixtures.
// It combines multiple sub-configurations to control the generation of different types of accounts,
// their balances, stakes, and other properties.
type MorseFixturesConfig struct {
	ValidAccountsConfig             // Configuration for valid accounts (EOAs, applications, validators)
	InvalidAccountsConfig           // Configuration for accounts with invalid addresses
	OrphanedActorsConfig            // Configuration for orphaned validators and applications
	UnbondingActorsConfig           // Configuration for unbonding validators and applications
	UnstakedAccountBalancesConfigFn // Configuration for unstaked account balances
	ValidatorStakesConfigFn         // Configuration for validator stake amounts
	ApplicationStakesConfigFn       // Configuration for application stake amounts
	ModuleAccountNameConfigFn       // Configuration for module account names
	UnstakingTimeConfig             // Configuration for unstaking times for actors which began unbonding on Morse
}

// GetTotalAccounts calculates the total number of accounts based on the configuration.
func (cfg *MorseFixturesConfig) GetTotalAccounts() uint64 {
	// Calculate the total number of accounts based on the configuration
	return cfg.NumAccountsValid +
		cfg.NumApplicationsValid +
		cfg.NumValidatorsValid +
		cfg.NumModuleAccounts +
		// 1 account for the operator and 1 for the owner (unstaked balances)
		(cfg.NumNonCustodialValidators * 2) +
		cfg.NumAddressTooShort +
		cfg.NumAddressTooLong +
		cfg.NumNonHexAddress +
		cfg.NumApplicationsOrphaned +
		cfg.NumValidatorsOrphaned +
		cfg.NumApplicationsUnbondingBegan +
		cfg.NumApplicationsUnbondingEnded +
		cfg.NumValidatorsUnbondingBegan +
		cfg.NumValidatorsUnbondingEnded
}

// UnstakedAccountBalancesConfigFn is a function that returns the balance for an unstaked
// account based on its index, actor type index, actor type, and existing account data.
type UnstakedAccountBalancesConfigFn func(
	index uint64, // The global index of the account
	actorTypeIndex uint64, // The index within the actor type group
	actorType MorseUnstakedActorType, // The type of the unstaked actor
	morseAccount *migrationtypes.MorseAccount, // The account to set the balance for
) *cosmostypes.Coin

// ModuleAccountNameConfigFn is a function that returns a module account name
// based on the global index and actor type index.
type ModuleAccountNameConfigFn func(index uint64, actorTypeIndex uint64) string

// ValidatorStakesConfigFn is a function that returns the staked and unstaked
// balances for a validator based on its index, actor type index, actor type,
// and existing validator data.
type ValidatorStakesConfigFn func(
	index uint64, // The global index of the validator
	actorTypeIndex uint64, // The index within the validator type group
	actorType MorseValidatorActorType, // The type of validator actor
	validator *migrationtypes.MorseValidator, // The validator to set balances for
) (staked, unstaked *cosmostypes.Coin)

// ApplicationStakesConfig is a function that returns the staked and unstaked
// balances for an application based on its index, actor type index, actor type,
// and existing application data.
type ApplicationStakesConfigFn func(
	index uint64, // The global index of the application
	actorTypeIndex uint64, // The index within the application type group
	actorType MorseApplicationActorType, // The type of application actor
	application *migrationtypes.MorseApplication, // The application to set balances for
) (staked, unstaked *cosmostypes.Coin)

// ValidAccountsConfig defines the number of valid accounts to generate
// for each account type in the test fixtures.
type ValidAccountsConfig struct {
	NumAccountsValid          uint64 // Number of regular externally owned accounts
	NumApplicationsValid      uint64 // Number of application accounts
	NumValidatorsValid        uint64 // Number of validator accounts
	NumNonCustodialValidators uint64 // Number of non-custodial validator accounts
	NumModuleAccounts         uint64 // Number of module accounts
}

// OrphanedActorsConfig defines the number of orphaned staked actors to generate.
// Orphaned actors have a staked position but no corresponding unstaked account.
type OrphanedActorsConfig struct {
	NumApplicationsOrphaned uint64 // Number of orphaned application accounts
	NumValidatorsOrphaned   uint64 // Number of orphaned validator accounts
}

// InvalidAccountsConfig defines the number of invalid accounts to generate
// with different types of address invalidity.
type InvalidAccountsConfig struct {
	NumAddressTooShort uint64 // Number of accounts with addresses that are too short
	NumAddressTooLong  uint64 // Number of accounts with addresses that are too long
	NumNonHexAddress   uint64 // Number of accounts with addresses containing non-hexadecimal characters
}

// UnbondingActorsConfig defines the number of unbonding and unbonded validators and applications to generate.
// DEV_NOTE: The accounts/actors are generated in the order they are defined in this struct.
type UnbondingActorsConfig struct {
	NumApplicationsUnbondingBegan uint64 // Number of applications to generate as having begun unbonding on Morse
	NumApplicationsUnbondingEnded uint64 // Number of applications to generate as having unbonded on Morse while waiting to be claimed
	NumValidatorsUnbondingBegan   uint64 // Number of validators to generate as having begun unbonding on Morse
	NumValidatorsUnbondingEnded   uint64 // Number of validators to generate as unbonded on Morse while waiting to be claimed
}

// UnstakingTimeConfig holds functions that determine the unstaking time for each actor type.
type UnstakingTimeConfig struct {
	ApplicationUnstakingTimeFn UnstakingTimeConfigFn[MorseApplicationActorType, *migrationtypes.MorseApplication]
	ValidatorUnstakingTimeFn   UnstakingTimeConfigFn[MorseValidatorActorType, *migrationtypes.MorseValidator]
}

// UnstakingTimeConfigFn defines a function that configures the unstaking time for an actor.
// The zero time.Time value (time.Time{}) indicates that the actor type is not unbonding/unbonded.
type UnstakingTimeConfigFn[T, A any] func(
	// The global index of the actor
	index uint64,
	// The index within the actor type group
	actorTypeIndex uint64,
	// The type of actor
	actorType T,
	// The actor to set the unstaking time for
	actor A,
) time.Time

// MorseFixturesOptionFn defines a function that configures a MorseFixturesConfig.
// This follows the functional options pattern for configuring structs.
type MorseFixturesOptionFn func(config *MorseFixturesConfig)

// WithModuleAccountNameFn sets the ModuleAccountNameConfig for the fixtures.
// It determines how module account names are generated during fixture creation.
func WithModuleAccountNameFn(nameFn ModuleAccountNameConfigFn) MorseFixturesOptionFn {
	return func(config *MorseFixturesConfig) {
		config.ModuleAccountNameConfigFn = nameFn
	}
}

// WithValidAccounts sets the ValidAccountsConfig for the fixtures.
// It determines how many valid accounts of each type are generated.
func WithValidAccounts(cfg ValidAccountsConfig) MorseFixturesOptionFn {
	return func(config *MorseFixturesConfig) {
		config.ValidAccountsConfig = cfg
	}
}

// WithInvalidAccounts sets the InvalidAccountsConfig for the fixtures.
// It determines how many invalid accounts of each type are generated.
func WithInvalidAccounts(cfg InvalidAccountsConfig) MorseFixturesOptionFn {
	return func(config *MorseFixturesConfig) {
		config.InvalidAccountsConfig = cfg
	}
}

// WithOrphanedActors sets the OrphanedActorsConfig for the fixtures.
// It determines how many orphaned applications and validators are generated.
func WithOrphanedActors(cfg OrphanedActorsConfig) MorseFixturesOptionFn {
	return func(config *MorseFixturesConfig) {
		config.OrphanedActorsConfig = cfg
	}
}

// WithUnstakedAccountBalancesFn sets the UnstakedAccountBalancesConfig for the fixtures.
// It defines how balances are determined for unstaked accounts.
func WithUnstakedAccountBalancesFn(balanceFn UnstakedAccountBalancesConfigFn) MorseFixturesOptionFn {
	return func(config *MorseFixturesConfig) {
		config.UnstakedAccountBalancesConfigFn = balanceFn
	}
}

// WithValidatorStakesFn sets the ValidatorStakesConfig for the fixtures.
// It defines how staked and unstaked balances are determined for validator accounts.
func WithValidatorStakesFn(stakeFn ValidatorStakesConfigFn) MorseFixturesOptionFn {
	return func(config *MorseFixturesConfig) {
		config.ValidatorStakesConfigFn = stakeFn
	}
}

// WithApplicationStakesFn sets the ApplicationStakesConfig for the fixtures.
// It defines how staked and unstaked balances are determined for application accounts.
func WithApplicationStakesFn(stakeFn ApplicationStakesConfigFn) MorseFixturesOptionFn {
	return func(config *MorseFixturesConfig) {
		config.ApplicationStakesConfigFn = stakeFn
	}
}

// WithUnbondingActors sets the UnbondingActorsConfig for the fixtures.
func WithUnbondingActors(cfg UnbondingActorsConfig) MorseFixturesOptionFn {
	return func(config *MorseFixturesConfig) {
		config.UnbondingActorsConfig = cfg
	}
}

// WithUnstakingTime sets the UnstakingTimeConfig for the fixtures.
func WithUnstakingTime(cfg UnstakingTimeConfig) MorseFixturesOptionFn {
	return func(config *MorseFixturesConfig) {
		config.UnstakingTimeConfig = cfg
	}
}

// NewMorseFixtures creates a new MorseMigrationFixtures instance with the provided options.
// It initializes the necessary data structures and generates test fixtures according to
// the configuration provided through the options.
func NewMorseFixtures(opts ...MorseFixturesOptionFn) (*MorseMigrationFixtures, error) {
	morseFixtures := &MorseMigrationFixtures{
		morseStateExport: &migrationtypes.MorseStateExport{
			AppState: &migrationtypes.MorseTendermintAppState{
				Application: &migrationtypes.MorseApplications{
					Applications: make([]*migrationtypes.MorseApplication, 0),
				},
				Auth: &migrationtypes.MorseAuth{
					Accounts: make([]*migrationtypes.MorseAuthAccount, 0),
				},
				Pos: &migrationtypes.MorsePos{
					Validators: make([]*migrationtypes.MorseValidator, 0),
				},
			},
		},
		morseAccountState: &migrationtypes.MorseAccountState{
			Accounts: make([]*migrationtypes.MorseClaimableAccount, 0),
		},
		actorTypeGroups: actorTypeGroups{
			unstakedAccounts: make(map[MorseUnstakedActorType][]*actorFixture[*migrationtypes.MorseAccount]),
			validators:       make(map[MorseValidatorActorType][]*actorFixture[*migrationtypes.MorseValidator]),
			applications:     make(map[MorseApplicationActorType][]*actorFixture[*migrationtypes.MorseApplication]),
		},
	}

	// Apply all the provided functional options to configure the fixtures
	morseFixtures.config = &MorseFixturesConfig{}
	for _, opt := range opts {
		opt(morseFixtures.config)
	}

	totalAccounts := morseFixtures.config.GetTotalAccounts()

	morseFixtures.morseAccountState.Accounts = make([]*migrationtypes.MorseClaimableAccount, totalAccounts)

	// Generate all the fixtures based on the configuration
	if err := morseFixtures.generate(); err != nil {
		return nil, err
	}

	return morseFixtures, nil
}

// GetConfig returns the configuration used for generating the fixtures.
func (mf *MorseMigrationFixtures) GetConfig() *MorseFixturesConfig {
	return mf.config
}

// GetMorseStateExport returns the generated Morse state export for migration testing.
func (mf *MorseMigrationFixtures) GetMorseStateExport() *migrationtypes.MorseStateExport {
	return mf.morseStateExport
}

// GetMorseAccountState returns the generated Morse account state for migration testing.
func (mf *MorseMigrationFixtures) GetMorseAccountState() *migrationtypes.MorseAccountState {
	return mf.morseAccountState
}

// GetUnstakedActorFixtures returns all unstaked actor fixtures of the specified account type.
// This method allows test code to access unstaked account fixtures (such as EOAs, module accounts,
// and accounts with invalid addresses) for test assertions and scenario setup.
func (mf *MorseMigrationFixtures) GetUnstakedActorFixtures(
	actorType MorseUnstakedActorType,
) []*actorFixture[*migrationtypes.MorseAccount] {
	return mf.actorTypeGroups.unstakedAccounts[actorType]
}

// GetValidatorFixtures returns all validator fixtures of the specified actor type.
// This method allows test code to access validator fixtures (such as standard validators
// with unstaked accounts or orphaned validators) for test assertions and scenario setup.
func (mf *MorseMigrationFixtures) GetValidatorFixtures(
	actorType MorseValidatorActorType,
) []*actorFixture[*migrationtypes.MorseValidator] {
	return mf.actorTypeGroups.validators[actorType]
}

// GetApplicationFixtures returns all application fixtures of the specified actor type.
// This method allows test code to access application fixtures (such as standard applications
// with unstaked accounts or orphaned applications) for test assertions and scenario setup.
func (mf *MorseMigrationFixtures) GetApplicationFixtures(
	actorType MorseApplicationActorType,
) []*actorFixture[*migrationtypes.MorseApplication] {
	return mf.actorTypeGroups.applications[actorType]
}

// nextAllAccountsIndex increments and returns the current index counter of the MorseMigrationFixtures.
// This method is used to generate sequential indices for various entities created during the
// fixture generation process, ensuring each entity has a unique index for deterministic generation.
func (mf *MorseMigrationFixtures) nextAllAccountsIndex() uint64 {
	mf.currentIndex++
	return mf.currentIndex - 1
}

// generate creates all the fixtures defined in the configuration.
// It systematically creates different types of accounts, applications, and validators,
// both valid and invalid, to be used in testing scenarios.
func (mf *MorseMigrationFixtures) generate() error {
	// Auth accounts section - Create unstaked accounts of various types

	// Generate valid regular externally owned accounts (EOAs)
	for i := range mf.config.NumAccountsValid {
		if _, err := mf.addAccount(i, MorseEOA); err != nil {
			return err
		}
	}

	// Generate module accounts which represent system accounts
	for i := range mf.config.NumModuleAccounts {
		if _, err := mf.addAccount(i, MorseModule); err != nil {
			return err
		}
	}

	// Generate invalid accounts with addresses that are too short
	for i := range mf.config.NumAddressTooShort {
		if _, err := mf.addAccount(i, MorseInvalidTooShort); err != nil {
			return err
		}
	}

	// Generate invalid accounts with addresses that are too long
	for i := range mf.config.NumAddressTooLong {
		if _, err := mf.addAccount(i, MorseInvalidTooLong); err != nil {
			return err
		}
	}

	// Generate accounts with invalid hex characters in the address
	for i := range mf.config.NumNonHexAddress {
		if _, err := mf.addAccount(i, MorseNonHex); err != nil {
			return err
		}
	}

	// Application accounts section - Create staked application accounts

	// Generate standard applications with both staked and unstaked accounts
	for i := range mf.config.NumApplicationsValid {
		if err := mf.addApplication(i, MorseApplication); err != nil {
			return err
		}
	}

	// Generate orphaned application accounts without corresponding unstaked accounts
	for i := range mf.config.NumApplicationsOrphaned {
		if err := mf.addApplication(i, MorseOrphanedApplication); err != nil {
			return err
		}
	}

	// Generate unbonding application accounts with both staked unstaked accounts
	for i := range mf.config.NumApplicationsUnbondingBegan {
		if err := mf.addApplication(i, MorseUnbondingApplication); err != nil {
			return err
		}
	}

	// Generate unbonded application accounts with both staked unstaked accounts
	for i := range mf.config.NumApplicationsUnbondingEnded {
		if err := mf.addApplication(i, MorseUnbondedApplication); err != nil {
			return err
		}
	}

	// Validator accounts section - Create staked validator accounts

	// Generate standard validators with both staked and unstaked accounts
	for i := range mf.config.NumValidatorsValid {
		if err := mf.addValidator(i, MorseValidator); err != nil {
			return err
		}
	}

	// Generate orphaned validator accounts without corresponding unstaked accounts
	for i := range mf.config.NumValidatorsOrphaned {
		if err := mf.addValidator(i, MorseOrphanedValidator); err != nil {
			return err
		}
	}

	// Generate non-custodial validators with both staked and unstaked operator accounts and unstaked owne
	for i := range mf.config.NumNonCustodialValidators {
		if _, err := mf.addAccount(i, MorseNonCustodialOwnerAccount); err != nil {
			return err
		}

		if err := mf.addValidator(i, MorseNonCustodialValidator); err != nil {
			return err
		}
	}

	// Generate unbonding validator accounts with both staked unstaked accounts
	for i := range mf.config.NumValidatorsUnbondingBegan {
		if err := mf.addValidator(i, MorseUnbondingValidator); err != nil {
			return err
		}
	}

	// Generate unbonded validator accounts with both staked unstaked accounts
	for i := range mf.config.NumValidatorsUnbondingEnded {
		if err := mf.addValidator(i, MorseUnbondedValidator); err != nil {
			return err
		}
	}

	return nil
}

// addAccount creates and adds an unstaked account to the Morse state export and account state.
// This function handles different types of accounts, including regular EOAs, module accounts,
// and accounts with invalid addresses for testing error handling.
func (mf *MorseMigrationFixtures) addAccount(
	actorTypeIndex uint64,
	unstakedActorType MorseUnstakedActorType,
) (unstakedActorFixture *actorFixture[*migrationtypes.MorseAccount], err error) {
	// Get the next global index for this account
	allAccountsIndex := mf.nextAllAccountsIndex()

	// Generate a deterministic private key for this account index
	privKey := mf.generateMorsePrivateKey(allAccountsIndex)
	pubKey := privKey.PubKey()
	address := pubKey.Address()

	// Determine the account type string and modify the address if needed based on the unstaked actor type enum.

	var accountType, moduleAccountName string
	switch unstakedActorType {
	case MorseEOA:
		accountType = migrationtypes.MorseExternallyOwnedAccountType
	case MorseModule:
		accountType = migrationtypes.MorseModuleAccountType
		// For module accounts, use a name-based address instead of a crypto address
		moduleAccountName = mf.config.ModuleAccountNameConfigFn(allAccountsIndex, actorTypeIndex)
	case MorseInvalidTooShort:
		accountType = migrationtypes.MorseExternallyOwnedAccountType
		// Create an invalid address that's too short
		address = address[:len(address)-1]
	case MorseInvalidTooLong:
		accountType = migrationtypes.MorseExternallyOwnedAccountType
		// Create an invalid address that's too long
		address = append(address, []byte{0x00}...)
	case MorseNonHex:
		accountType = migrationtypes.MorseExternallyOwnedAccountType
		// Create an invalid address with non-hexadecimal characters
		invalidBytes := []byte("invalidhex_")
		address = append(invalidBytes, address[len(invalidBytes):]...)
	}

	// Create the Morse account with the address and public key
	morseAccount := &migrationtypes.MorseAccount{
		Address: address,
		PubKey: &migrationtypes.MorsePublicKey{
			Value: pubKey.Bytes(),
		},
	}

	// Set the account's balance based on the configuration
	balance := mf.config.UnstakedAccountBalancesConfigFn(
		allAccountsIndex,
		actorTypeIndex,
		unstakedActorType,
		morseAccount,
	)
	morseAccount.Coins = cosmostypes.NewCoins(*balance)

	var morseAccountJSONBz []byte
	// Module accounts need to be marshaled differently because they have additional properties
	if unstakedActorType == MorseModule {
		// Create a module account structure with name and permissions
		morseModuleAccount := &migrationtypes.MorseModuleAccount{
			Name:        moduleAccountName,
			BaseAccount: *morseAccount,
		}
		morseAccountJSONBz, err = cmtjson.Marshal(morseModuleAccount)
	} else {
		// For non-module accounts, marshal the account directly
		morseAccountJSONBz, err = cmtjson.Marshal(morseAccount)
	}
	if err != nil {
		return nil, err
	}

	// Add the account to the Morse state export with the appropriate type
	mf.morseStateExport.AppState.Auth.Accounts = append(
		mf.morseStateExport.AppState.Auth.Accounts,
		&migrationtypes.MorseAuthAccount{
			Type:  accountType,
			Value: morseAccountJSONBz,
		},
	)

	// Convert the account to a claimable account for migration
	morseClaimableAccount, err := mf.generateMorseClaimableAccount(morseAccount)
	if err != nil {
		return nil, err
	}

	// Store the claimable account in the account state
	mf.morseAccountState.Accounts[allAccountsIndex] = morseClaimableAccount

	// Create an unstaked account actorFixture to track this account and add it to the appropriate
	// group in the fixtures to make it available for test assertions and scenario setup.
	unstakedActorFixture = &actorFixture[*migrationtypes.MorseAccount]{
		actor:            morseAccount,
		claimableAccount: morseClaimableAccount,
		privKey:          privKey,
	}

	mf.actorTypeGroups.unstakedAccounts[unstakedActorType] = append(
		mf.actorTypeGroups.unstakedAccounts[unstakedActorType],
		unstakedActorFixture,
	)

	return unstakedActorFixture, nil
}

// addApplication creates and adds a staked application account to the Morse state export
// and account state. This function handles both standard applications that have unstaked
// balances associated with the same address and "orphaned" applications, which don't.
func (mf *MorseMigrationFixtures) addApplication(
	actorIndex uint64,
	applicationType MorseApplicationActorType,
) error {
	// Get the next global index for this application
	allAccountsIndex := mf.nextAllAccountsIndex()

	// Generate a deterministic private key for this application
	privKey := mf.generateMorsePrivateKey(allAccountsIndex)
	pubKey := privKey.PubKey()

	// Create a new MorseApplication with basic properties
	morseApplication := &migrationtypes.MorseApplication{
		Address:   pubKey.Address(),
		PublicKey: pubKey.Bytes(),
		Jailed:    false,
		Status:    2, // Status 2 represents a bonded/active application
	}

	// Get the staked and unstaked balances for this application from the configuration
	if mf.config.ApplicationStakesConfigFn == nil {
		panic("ApplicationStakesConfigFn is required when using ValidAccountsConfig with non-zero NumApplications")
	}
	stakedBalance, unstakedBalance := mf.config.ApplicationStakesConfigFn(
		allAccountsIndex,
		actorIndex,
		applicationType,
		morseApplication,
	)
	morseApplication.StakedTokens = fmt.Sprintf("%d", stakedBalance.Amount.Int64())

	// Add the application to the Morse state export
	mf.morseStateExport.AppState.Application.Applications = append(
		mf.morseStateExport.AppState.Application.Applications,
		morseApplication,
	)

	// Convert to a claimable account for migration
	morseClaimableAccount, err := mf.generateMorseClaimableAccount(morseApplication)
	if err != nil {
		return err
	}

	switch applicationType {
	// Set the unstaking time for unbonding and unbonded applications
	case MorseUnbondingApplication, MorseUnbondedApplication:
		if mf.config.ApplicationUnstakingTimeFn == nil {
			panic("UnstakingTimeConfigFn is required when using UnbondingActorsConfig")
		}

		unstakingTime := mf.config.ApplicationUnstakingTimeFn(
			allAccountsIndex,
			actorIndex,
			applicationType,
			morseApplication,
		)
		morseClaimableAccount.UnstakingTime = unstakingTime
	}

	// Store the claimable application in the account state
	mf.morseAccountState.Accounts[allAccountsIndex] = morseClaimableAccount

	// For non-orphaned applications, also create an unstaked account counterpart
	// Orphaned applications don't have corresponding unstaked accounts, so we skip this step
	if applicationType != MorseOrphanedApplication {
		if err = mf.addUnstakedAccountForStakedActor(allAccountsIndex, pubKey, unstakedBalance); err != nil {
			return err
		}
	}

	// Create an application actorFixture to track this account and add it to the appropriate
	// group in the fixtures to make it available for test assertions and scenario setup.
	morseApplicationFixture := &actorFixture[*migrationtypes.MorseApplication]{
		actor:            morseApplication,
		claimableAccount: morseClaimableAccount,
		privKey:          privKey,
	}

	mf.actorTypeGroups.applications[applicationType] = append(
		mf.actorTypeGroups.applications[applicationType],
		morseApplicationFixture,
	)

	return nil
}

// addValidator creates and adds a staked validator account to the Morse state export
// and account state. This function handles both standard validators with unstaked
// counterparts and orphaned validators without unstaked accounts.
func (mf *MorseMigrationFixtures) addValidator(
	actorIndex uint64,
	validatorType MorseValidatorActorType,
) error {
	// Get the next global index for this validator
	allAccountsIndex := mf.nextAllAccountsIndex()

	// Generate a deterministic private key for this validator
	privKey := mf.generateMorsePrivateKey(allAccountsIndex)
	pubKey := privKey.PubKey()

	// Create a new MorseValidator with basic properties
	morseValidator := &migrationtypes.MorseValidator{
		Address:   pubKey.Address(),
		PublicKey: pubKey.Bytes(),
		Jailed:    false,
		Status:    2, // Status 2 represents a bonded/active validator
	}

	// Get the staked and unstaked balances for this validator from the configuration
	if mf.config.ValidatorStakesConfigFn == nil {
		panic("ValidatorStakesConfigFn is required when using ValidValidatorConfig with non-zero NumValidators")
	}
	stakedBalance, unstakedBalance := mf.config.ValidatorStakesConfigFn(
		allAccountsIndex,
		actorIndex,
		validatorType,
		morseValidator,
	)
	morseValidator.StakedTokens = fmt.Sprintf("%d", stakedBalance.Amount.Int64())

	// Add the validator to the Morse state export in the proof-of-stake (pos) module
	mf.morseStateExport.AppState.Pos.Validators = append(
		mf.morseStateExport.AppState.Pos.Validators,
		morseValidator,
	)

	// Convert to a claimable account for migration
	morseClaimableAccount, err := mf.generateMorseClaimableAccount(morseValidator)
	if err != nil {
		return err
	}

	switch validatorType {

	// For orphaned validators, no unstaked account is needed; bypass the default case.
	case MorseOrphanedValidator:
		// No-op

	// For non-custodial validators, also create an unstaked owner account.
	case MorseNonCustodialValidator:
		// DEV_NOTE: IMPLICIT ASSUMPTION that the previous account index belongs to the Morse owner (unstaked) account.
		// This is a safe assumption because the Morse owner account is always created before the Morse validator account.
		morseOwnerPrivKey := mf.generateMorsePrivateKey(allAccountsIndex - 1)
		morseClaimableAccount.MorseOutputAddress = morseOwnerPrivKey.PubKey().Address().String()

	// For unbonding and unbonded suppliers, set the unstaking time.
	case MorseUnbondingValidator, MorseUnbondedValidator:
		if mf.config.ValidatorUnstakingTimeFn == nil {
			panic("UnstakingTimeConfigFn is required when using UnbondingActorsConfig")
		}

		if mf.config.ValidatorUnstakingTimeFn != nil {
			unstakingTime := mf.config.ValidatorUnstakingTimeFn(
				allAccountsIndex,
				actorIndex,
				validatorType,
				morseValidator,
			)
			morseClaimableAccount.UnstakingTime = unstakingTime
		}
	}

	// Store the claimable validator in the account state.
	mf.morseAccountState.Accounts[allAccountsIndex] = morseClaimableAccount

	// For non-orphaned validators, create an unstaked account counterpart.
	// Orphaned validators don't have corresponding unstaked accounts, so we skip this step
	if validatorType != MorseOrphanedValidator {
		if err = mf.addUnstakedAccountForStakedActor(allAccountsIndex, pubKey, unstakedBalance); err != nil {
			return err
		}
	}

	// Create a validator actorFixture to track this account and add it to the appropriate
	// group in the fixtures to make it available for test assertions and scenario setup.
	validatorFixture := &actorFixture[*migrationtypes.MorseValidator]{
		actor:            morseValidator,
		claimableAccount: morseClaimableAccount,
		privKey:          privKey,
	}

	mf.actorTypeGroups.validators[validatorType] = append(
		mf.actorTypeGroups.validators[validatorType],
		validatorFixture,
	)

	return nil
}

// addUnstakedAccountForStakedActor creates and adds an unstaked account that corresponds to a
// staked actor (validator or application) to the Morse state export and account state.
// This function is used to create the unstaked side of a staked actor (which has both a staked
// and unstaked representation in the blockchain).
func (mf *MorseMigrationFixtures) addUnstakedAccountForStakedActor(
	allAccountsIndex uint64,
	pubKey crypto.PubKey,
	unstakedBalance *cosmostypes.Coin,
) error {
	// Create a new MorseAccount with the public key and balance
	address := pubKey.Address()
	morseAccount := &migrationtypes.MorseAccount{
		Address: address,
		PubKey: &migrationtypes.MorsePublicKey{
			Value: pubKey.Bytes(),
		},
		Coins: cosmostypes.NewCoins(*unstakedBalance),
	}

	// Marshal the account to JSON for storage in the state export
	morseAccountJSONBz, err := cmtjson.Marshal(morseAccount)
	if err != nil {
		return err
	}

	// Add the account to the Morse state export as an externally owned account (EOA)
	mf.morseStateExport.AppState.Auth.Accounts = append(
		mf.morseStateExport.AppState.Auth.Accounts,
		&migrationtypes.MorseAuthAccount{
			Type:  migrationtypes.MorseExternallyOwnedAccountType,
			Value: morseAccountJSONBz,
		},
	)

	// Store the claimable account in the account state
	mf.morseAccountState.Accounts[allAccountsIndex].UnstakedBalance = *unstakedBalance

	return nil
}

// generateMorsePrivateKey generates a deterministic private key for the given index
// and tracks it in the MorseMigrationFixtures instance.
func (mf *MorseMigrationFixtures) generateMorsePrivateKey(seed uint64) cometcrypto.PrivKey {
	seedBz := make([]byte, 8)
	binary.LittleEndian.PutUint64(seedBz, seed)

	return cometcrypto.GenPrivKeyFromSecret(seedBz)
}

// generateMorseClaimableAccount creates a MorseClaimableAccount from a MorseAccount
// with default settings for a basic account (not staked as a validator or application).
// This function is used for regular EOA accounts that don't have any staked tokens.
func (mf *MorseMigrationFixtures) generateMorseClaimableAccount(
	morseAccount any,
) (*migrationtypes.MorseClaimableAccount, error) {
	if morseAccount == nil {
		return nil, fmt.Errorf("morse account is nil")
	}

	morseClaimableAccount := &migrationtypes.MorseClaimableAccount{
		UnstakedBalance:  cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 0),
		ApplicationStake: cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 0),
		SupplierStake:    cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 0),
	}
	switch account := morseAccount.(type) {
	case *migrationtypes.MorseAccount:
		morseClaimableAccount.MorseSrcAddress = account.Address.String()
		morseClaimableAccount.UnstakedBalance.Amount = account.Coins[0].Amount
	case *migrationtypes.MorseValidator:
		morseClaimableAccount.MorseSrcAddress = account.Address.String()
		stakedTokensInt, err := strconv.ParseInt(account.StakedTokens, 10, 64)
		if err != nil {
			return nil, err
		}
		morseClaimableAccount.SupplierStake.Amount = math.NewInt(stakedTokensInt)
	case *migrationtypes.MorseApplication:
		morseClaimableAccount.MorseSrcAddress = account.Address.String()
		stakedTokensInt, err := strconv.ParseInt(account.StakedTokens, 10, 64)
		if err != nil {
			return nil, err
		}
		morseClaimableAccount.ApplicationStake.Amount = math.NewInt(stakedTokensInt)
	default:
		return nil, fmt.Errorf("unsupported morse account type: %T", account)
	}

	return morseClaimableAccount, nil
}
