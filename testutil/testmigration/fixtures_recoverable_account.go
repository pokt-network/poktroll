package testmigration

import (
	"encoding/binary"
	"fmt"
	"strconv"

	"cosmossdk.io/math"
	"github.com/cometbft/cometbft/crypto"
	cometcrypto "github.com/cometbft/cometbft/crypto/ed25519"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/app/volatile"
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
)

// MorseValidatorActorType represents different types of validator actors
// in Morse that can be included in test fixtures.
type MorseValidatorActorType int

const (
	// MorseValidator represents a standard validator with both staked and unstaked accounts
	MorseValidator = MorseValidatorActorType(iota)
	// MorseOrphanedValidator represents a validator without a corresponding unstaked account
	MorseOrphanedValidator
)

// MorseApplicationActorType represents different types of application actors
// in Morse that can be included in test fixtures.
type MorseApplicationActorType int

const (
	// MorseApplication represents a standard application with both staked and unstaked accounts
	MorseApplication = MorseApplicationActorType(iota)
	// MorseOrphanedApplication represents an application without a corresponding unstaked account
	MorseOrphanedApplication
)

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
}

// MorseFixturesConfig defines the configuration parameters for generating Morse migration test fixtures.
// It combines multiple sub-configurations to control the generation of different types of accounts,
// their balances, stakes, and other properties.
type MorseFixturesConfig struct {
	ValidAccountsConfig             // Configuration for valid accounts (EOAs, applications, validators)
	InvalidAccountsConfig           // Configuration for accounts with invalid addresses
	OrphanedActorsConfig            // Configuration for orphaned validators and applications
	UnstakedAccountBalancesConfigFn // Configuration for unstaked account balances
	ValidatorStakesConfigFn         // Configuration for validator stake amounts
	ApplicationStakesConfigFn       // Configuration for application stake amounts
	ModuleAccountNameConfigFn       // Configuration for module account names
}

// GetTotalAccounts calculates the total number of accounts based on the configuration.
func (cfg *MorseFixturesConfig) GetTotalAccounts() uint64 {
	// Calculate the total number of accounts based on the configuration
	return cfg.ValidAccountsConfig.NumAccounts +
		cfg.ValidAccountsConfig.NumApplications +
		cfg.ValidAccountsConfig.NumValidators +
		cfg.ValidAccountsConfig.NumModuleAccounts +
		cfg.InvalidAccountsConfig.NumAddressTooShort +
		cfg.InvalidAccountsConfig.NumAddressTooLong +
		cfg.InvalidAccountsConfig.NumNonHexAddress +
		cfg.OrphanedActorsConfig.NumApplications +
		cfg.OrphanedActorsConfig.NumValidators
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
	NumAccounts       uint64 // Number of regular externally owned accounts
	NumApplications   uint64 // Number of application accounts
	NumValidators     uint64 // Number of validator accounts
	NumModuleAccounts uint64 // Number of module accounts
}

// OrphanedActorsConfig defines the number of orphaned staked actors to generate.
// Orphaned actors have a staked position but no corresponding unstaked account.
type OrphanedActorsConfig struct {
	NumApplications uint64 // Number of orphaned application accounts
	NumValidators   uint64 // Number of orphaned validator accounts
}

// InvalidAccountsConfig defines the number of invalid accounts to generate
// with different types of address invalidity.
type InvalidAccountsConfig struct {
	NumAddressTooShort uint64 // Number of accounts with addresses that are too short
	NumAddressTooLong  uint64 // Number of accounts with addresses that are too long
	NumNonHexAddress   uint64 // Number of accounts with addresses containing non-hexadecimal characters
}

// MorseFixturesOptionFn defines a function that configures a MorseFixturesConfig.
// This follows the functional options pattern for configuring structs.
type MorseFixturesOptionFn func(config *MorseFixturesConfig)

// WithModuleAccountNameFn sets the ModuleAccountNameConfig for the fixtures.
// It determines how module account names are generated during fixture creation.
func WithModuleAccountNameFn(cfg ModuleAccountNameConfigFn) MorseFixturesOptionFn {
	return func(config *MorseFixturesConfig) {
		config.ModuleAccountNameConfigFn = cfg
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
func WithUnstakedAccountBalancesFn(cfg UnstakedAccountBalancesConfigFn) MorseFixturesOptionFn {
	return func(config *MorseFixturesConfig) {
		config.UnstakedAccountBalancesConfigFn = cfg
	}
}

// WithValidatorStakesFn sets the ValidatorStakesConfig for the fixtures.
// It defines how staked and unstaked balances are determined for validator accounts.
func WithValidatorStakesFn(cfg ValidatorStakesConfigFn) MorseFixturesOptionFn {
	return func(config *MorseFixturesConfig) {
		config.ValidatorStakesConfigFn = cfg
	}
}

// WithApplicationStakesFn sets the ApplicationStakesConfig for the fixtures.
// It defines how staked and unstaked balances are determined for application accounts.
func WithApplicationStakesFn(cfg ApplicationStakesConfigFn) MorseFixturesOptionFn {
	return func(config *MorseFixturesConfig) {
		config.ApplicationStakesConfigFn = cfg
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

// nextAllAccountsIndex increments and returns the current index counter of the MorseMigrationFixtures.
// This method is used to generate sequential indices for various entities created during the
// fixture generation process, ensuring each entity has a unique index for deterministic generation.
func (mf *MorseMigrationFixtures) nextAllAccountsIndex() uint64 {
	mf.currentIndex++
	return mf.currentIndex - 1
}

// GetMorseStateExport returns the generated Morse state export for migration testing.
func (mf *MorseMigrationFixtures) GetMorseStateExport() *migrationtypes.MorseStateExport {
	return mf.morseStateExport
}

// GetMorseAccountState returns the generated Morse account state for migration testing.
func (mf *MorseMigrationFixtures) GetMorseAccountState() *migrationtypes.MorseAccountState {
	return mf.morseAccountState
}

// generate creates all the fixtures defined in the configuration.
// It systematically creates different types of accounts, applications, and validators,
// both valid and invalid, to be used in testing scenarios.
func (mf *MorseMigrationFixtures) generate() error {
	// Auth accounts section - Create unstaked accounts of various types

	// Generate valid regular externally owned accounts (EOAs)
	for i := range mf.config.ValidAccountsConfig.NumAccounts {
		if err := mf.addAccount(i, MorseEOA); err != nil {
			return err
		}
	}

	// Generate module accounts which represent system accounts
	for i := range mf.config.ValidAccountsConfig.NumModuleAccounts {
		if err := mf.addAccount(i, MorseModule); err != nil {
			return err
		}
	}

	// Generate invalid accounts with addresses that are too short
	for i := range mf.config.InvalidAccountsConfig.NumAddressTooShort {
		if err := mf.addAccount(i, MorseInvalidTooShort); err != nil {
			return err
		}
	}

	// Generate invalid accounts with addresses that are too long
	for i := range mf.config.InvalidAccountsConfig.NumAddressTooLong {
		if err := mf.addAccount(i, MorseInvalidTooLong); err != nil {
			return err
		}
	}

	// Generate accounts with invalid hex characters in the address
	for i := range mf.config.InvalidAccountsConfig.NumNonHexAddress {
		if err := mf.addAccount(i, MorseNonHex); err != nil {
			return err
		}
	}

	// Application accounts section - Create staked application accounts

	// Generate standard applications with both staked and unstaked accounts
	for i := range mf.config.ValidAccountsConfig.NumApplications {
		if err := mf.addApplication(i, MorseApplication); err != nil {
			return err
		}
	}

	// Generate orphaned application accounts without corresponding unstaked accounts
	for i := range mf.config.OrphanedActorsConfig.NumApplications {
		if err := mf.addApplication(i, MorseOrphanedApplication); err != nil {
			return err
		}
	}

	// Validator accounts section - Create staked validator accounts

	// Generate standard validators with both staked and unstaked accounts
	for i := range mf.config.ValidAccountsConfig.NumValidators {
		if err := mf.addValidator(i, MorseValidator); err != nil {
			return err
		}
	}

	// Generate orphaned validator accounts without corresponding unstaked accounts
	for i := range mf.config.OrphanedActorsConfig.NumValidators {
		if err := mf.addValidator(i, MorseOrphanedValidator); err != nil {
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
) (err error) {
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
		return err
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
		return err
	}

	// Store the claimable account in the account state
	mf.morseAccountState.Accounts[allAccountsIndex] = morseClaimableAccount

	return nil
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

	// Store the claimable application in the account state
	mf.morseAccountState.Accounts[allAccountsIndex] = morseClaimableAccount

	// For non-orphaned applications, also create an unstaked account counterpart
	if applicationType != MorseOrphanedApplication {
		if err := mf.addUnstakedAccountForStakedActor(allAccountsIndex, pubKey, unstakedBalance); err != nil {
			return err
		}
	}
	// Orphaned applications don't have corresponding unstaked accounts, so we skip this step

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

	// Store the claimable validator in the account state
	mf.morseAccountState.Accounts[allAccountsIndex] = morseClaimableAccount

	// For non-orphaned validators, also create an unstaked account counterpart
	if validatorType != MorseOrphanedValidator {
		if err := mf.addUnstakedAccountForStakedActor(allAccountsIndex, pubKey, unstakedBalance); err != nil {
			return err
		}
	}
	// Orphaned validators don't have corresponding unstaked accounts, so we skip this step

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
		UnstakedBalance:  cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 0),
		ApplicationStake: cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 0),
		SupplierStake:    cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 0),
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
