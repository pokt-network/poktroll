package testmigration

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"testing"

	cometcrypto "github.com/cometbft/cometbft/crypto/ed25519"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/volatile"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

// MorseAccountActorType is an enum which represents all possible staked and
// unstaked actor types which are considered in the migration module.
type MorseAccountActorType int

const (
	MorseUnstakedActor = MorseAccountActorType(iota)
	MorseApplicationActor
	MorseSupplierActor

	// NumMorseAccountActorTypes is the number of MorseAccountActorTypes.
	// It takes advantage of the fact that the enum is zero-indexed.
	NumMorseAccountActorTypes
)

type MorseUnstakedActorType int

const (
	MorseEOA = MorseUnstakedActorType(iota)
	MorseInvalidTooShort
	MorseInvalidTooLong
	MorseNonHex
	MorseModule
)

type MorseSupplierActorType int

const (
	MorseSupplier = MorseSupplierActorType(iota)
	MorseOrphanedSupplier
)

type MorseApplicationActorType int

const (
	MorseApplication = MorseApplicationActorType(iota)
	MorseOrphanedApplication
)

// MorseAccountActorTypeDistributionFn is a function which returns a MorseAccountActorType
// derived from the given index. It is intended to be used in conjunction with
// MorseStateExport and MorseAccountState fixture generation logic.
type MorseAccountActorTypeDistributionFn func(index uint64) MorseAccountActorType

// RoundRobinAllMorseAccountActorTypes cyclically returns each MorseAccountActorType, one after the other, as the index increases.
// It is used to map a test account index to the test actor that's generated.
func RoundRobinAllMorseAccountActorTypes(index uint64) MorseAccountActorType {
	return MorseAccountActorType(index % uint64(NumMorseAccountActorTypes))
}

// NewRoundRobinClusteredAllMorseAccountActorTypes returns a function which returns a MorseAccountActorType
// for every index, cyclically returning "clusters" of each MorseAccountActorType; i.e. continuous account types of clusterSize.
// E.g. if clusterSize is 2, then the resulting actor type sequence would be:
// - 0: MorseUnstakedActor
// - 1: MorseUnstakedActor
// - 2: MorseApplicationActor
// - 3: MorseApplicationActor
// - 4: MorseSupplierActor
// - 5: MorseSupplierActor
func NewRoundRobinClusteredAllMorseAccountActorTypes(clusterSize uint64) func(index uint64) MorseAccountActorType {
	index := uint64(0)
	actorTypeIndex := uint64(0)
	return func(_ uint64) MorseAccountActorType {
		if index >= clusterSize {
			index = 0
			actorTypeIndex++
		}

		if actorTypeIndex >= uint64(NumMorseAccountActorTypes) {
			actorTypeIndex = 0
		}

		return MorseAccountActorType(index % clusterSize)
	}
}

// AllUnstakedMorseAccountActorType returns MorseUnstakedActor for every index.
func AllUnstakedMorseAccountActorType(index uint64) MorseAccountActorType {
	return NewSingleMorseAccountActorTypeFn(MorseUnstakedActor)(index)
}

// AllApplicationMorseAccountActorType returns MorseApplicationActor for every index.
func AllApplicationMorseAccountActorType(index uint64) MorseAccountActorType {
	return NewSingleMorseAccountActorTypeFn(MorseApplicationActor)(index)
}

// AllSupplierMorseAccountActorType returns MorseSupplierActor for every index.
func AllSupplierMorseAccountActorType(index uint64) MorseAccountActorType {
	return NewSingleMorseAccountActorTypeFn(MorseSupplierActor)(index)
}

// NewSingleMorseAccountActorTypeFn returns a MorseAccountActorTypeDistributionFn
// which returns the given actor type for every index.
func NewSingleMorseAccountActorTypeFn(actorType MorseAccountActorType) MorseAccountActorTypeDistributionFn {
	return func(_ uint64) MorseAccountActorType {
		return actorType
	}
}

// GetRoundRobinMorseAccountActorType returns the actor type for the given index,
// given a round-robin distribution.
func GetRoundRobinMorseAccountActorType(idx uint64) MorseAccountActorType {
	return MorseAccountActorType(idx % uint64(NumMorseAccountActorTypes))
}

// TODO_IN_THIS_COMMIT: move & godoc...
type MorseMigrationFixtures struct {
	config            MorseFixturesConfig
	morseStateExport  *migrationtypes.MorseStateExport
	morseAccountState *migrationtypes.MorseAccountState

	morseKeysByIndex      map[uint64]cometcrypto.PrivKey
	morseKeyIndexexByAddr map[string]uint64
	morseKeysByAddr       map[string]cometcrypto.PrivKey
}

// TODO_IN_THIS_COMMIT: move & godoc...
type MorseFixturesConfig struct {
	ValidAccountsConfig
	InvalidAccountsConfig
	OrphanedActorsConfig
	UnstakedAccountBalancesConfig
	SupplierStakesConfig
	ApplicationStakesConfig
}

// TODO_IN_THIS_COMMIT: move & godoc...
type UnstakedAccountBalancesConfig struct {
	GetBalance func(
		index uint64,
		actorTypeIndex uint64,
		actorType MorseUnstakedActorType,
		morseAccount *migrationtypes.MorseAccount,
	) *cosmostypes.Coin
}

type SupplierStakesConfig struct {
	GetStakedAndUnstakedBalances func(
		index uint64,
		actorTypeIndex uint64,
		actorType MorseSupplierActorType,
		supplier *migrationtypes.MorseValidator,
	) (staked, unstaked *cosmostypes.Coin)
}

type ApplicationStakesConfig struct {
	GetStakedAndUnstakedBalances func(
		index uint64,
		actorTypeIndex uint64,
		actorType MorseApplicationActorType,
		application *migrationtypes.MorseApplication,
	) (staked, unstaked *cosmostypes.Coin)
}

// TODO_IN_THIS_COMMIT: move & godoc...
func (mf *MorseMigrationFixtures) GetConfig() MorseFixturesConfig {
	return mf.config
}

// TODO_IN_THIS_COMMIT: move & godoc...
type ValidAccountsConfig struct {
	NumAccounts       uint64
	NumApplications   uint64
	NumSuppliers      uint64
	NumModuleAccounts uint64
}

// TODO_IN_THIS_COMMIT: move & godoc...
// ... don't have corresponding MorseAuthAccounts ... happens in REAL (mainnet) snapshot data.
type OrphanedActorsConfig struct {
	NumApplications uint64
	NumSuppliers    uint64
}

// TODO_IN_THIS_COMMIT: move & godoc...
type InvalidAccountsConfig struct {
	NumAddressTooShort uint64
	NumAddressTooLong  uint64
	NumInvalidHex      uint64
}

// TODO_IN_THIS_COMMIT: move & godoc...
type MorseFixturesOption func(config MorseFixturesConfig)

// TODO_IN_THIS_COMMIT: move & godoc...
func WithValidAccounts(cfg ValidAccountsConfig) MorseFixturesOption {
	return func(config MorseFixturesConfig) {
		config.ValidAccountsConfig = cfg
	}
}

// TODO_IN_THIS_COMMIT: move & godoc...
func WithInvalidAccounts(cfg InvalidAccountsConfig) MorseFixturesOption {
	return func(config MorseFixturesConfig) {
		config.InvalidAccountsConfig = cfg
	}
}

// TODO_IN_THIS_COMMIT: move & godoc...
func WithOrphanedActors(cfg OrphanedActorsConfig) MorseFixturesOption {
	return func(config MorseFixturesConfig) {
		config.OrphanedActorsConfig = cfg
	}
}

func WithUnstakedAccountBalances(cfg UnstakedAccountBalancesConfig) MorseFixturesOption {
	return func(config MorseFixturesConfig) {
		config.UnstakedAccountBalancesConfig = cfg
	}
}

func WithSupplierStakes(cfg SupplierStakesConfig) MorseFixturesOption {
	return func(config MorseFixturesConfig) {
		config.SupplierStakesConfig = cfg
	}
}

func WithApplicationStakes(cfg ApplicationStakesConfig) MorseFixturesOption {
	return func(config MorseFixturesConfig) {
		config.ApplicationStakesConfig = cfg
	}
}

// TODO_IN_THIS_COMMIT: move & godoc...
func (mf *MorseMigrationFixtures) GetMorseStateExport() *migrationtypes.MorseStateExport {
	return mf.morseStateExport
}

// TODO_IN_THIS_COMMIT: move & godoc...
func (mf *MorseMigrationFixtures) GetMorseAccountState() *migrationtypes.MorseAccountState {
	return mf.morseAccountState
}

// TODO_IN_THIS_COMMIT: move & godoc...
func NewMorseFixtures(opts ...MorseFixturesOption) (*MorseMigrationFixtures, error) {
	morseFixtures := &MorseMigrationFixtures{
		morseStateExport:  new(migrationtypes.MorseStateExport),
		morseAccountState: new(migrationtypes.MorseAccountState),
	}

	morseFixtures.config = MorseFixturesConfig{}
	for _, opt := range opts {
		opt(morseFixtures.config)
	}

	if err := morseFixtures.generate(); err != nil {
		return nil, err
	}

	return morseFixtures, nil
}

// TODO_IN_THIS_COMMIT: move & godoc...
func (mf *MorseMigrationFixtures) generate() error {
	allActorsIndex := uint64(0)

	// Auth accounts

	for i := range mf.config.ValidAccountsConfig.NumAccounts {
		if err := mf.addAccount(allActorsIndex, i, MorseEOA); err != nil {
			return err
		}
		allActorsIndex++
	}

	for i := range mf.config.ValidAccountsConfig.NumModuleAccounts {
		if err := mf.addAccount(allActorsIndex, i, MorseModule); err != nil {
			return err
		}
		allActorsIndex++
	}

	for i := range mf.config.InvalidAccountsConfig.NumAddressTooShort {
		if err := mf.addAccount(allActorsIndex, i, MorseInvalidTooShort); err != nil {
			return err
		}
		allActorsIndex++
	}

	for i := range mf.config.InvalidAccountsConfig.NumAddressTooLong {
		if err := mf.addAccount(allActorsIndex, i, MorseInvalidTooLong); err != nil {
			return err
		}
		allActorsIndex++
	}

	for i := range mf.config.InvalidAccountsConfig.NumInvalidHex {
		if err := mf.addAccount(allActorsIndex, i, MorseNonHex); err != nil {
			return err
		}
		allActorsIndex++
	}

	// Application accounts

	for i := range mf.config.ValidAccountsConfig.NumApplications {
		if err := mf.addApplication(allActorsIndex, i, MorseApplication); err != nil {
			return err
		}
		allActorsIndex++
	}

	for i := range mf.config.OrphanedActorsConfig.NumApplications {
		if err := mf.addApplication(allActorsIndex, i, MorseOrphanedApplication); err != nil {
			return err
		}
		allActorsIndex++
	}

	// Supplier accounts

	for i := range mf.config.ValidAccountsConfig.NumSuppliers {
		if err := mf.addSupplier(allActorsIndex, i, MorseSupplier); err != nil {
			return err
		}
		allActorsIndex++
	}

	for i := range mf.config.OrphanedActorsConfig.NumSuppliers {
		if err := mf.addSupplier(allActorsIndex, i, MorseOrphanedSupplier); err != nil {
			return err
		}
		allActorsIndex++
	}

	return nil
}

// TODO_IN_THIS_COMMIT: move & godoc...
func (mf *MorseMigrationFixtures) addAccount(
	allActorsIndex,
	actorTypeIndex uint64,
	accountType MorseUnstakedActorType,
) error {
	morseAccount, err := mf.GenMorseAccount(allActorsIndex, actorTypeIndex, accountType)
	if err != nil {
		return err
	}

	var actorType string
	var morseAccountJSONBz []byte
	switch accountType {
	case MorseModule:
		// Marshal module
		actorType = migrationtypes.MorseModuleAccountType
	default:
		actorType = migrationtypes.MorseExternallyOwnedAccountType
	}

	if accountType == MorseModule {
		// unmarshal module
	} else {
		morseAccountJSONBz, err = cmtjson.Marshal(morseAccount)
		if err != nil {
			return err
		}
	}

	mf.morseStateExport.AppState.Auth.Accounts = append(
		mf.morseStateExport.AppState.Auth.Accounts,
		&migrationtypes.MorseAuthAccount{
			Type:  actorType,
			Value: morseAccountJSONBz,
		},
	)

	// Add the account to the morseAccountState.
	morseClaimableAccount, err := mf.AddMorseClaimableAccount(morseAccount)
	if err != nil {
		return err
	}

	mf.morseAccountState.Accounts[allActorsIndex] = morseClaimableAccount

	return nil
}

func (mf *MorseMigrationFixtures) addApplication(
	allActorsIndex,
	actorIndex uint64,
	applicationType MorseApplicationActorType,
) error {
	morseApplication, unstakedBalance, err := mf.GenMorseApplication(allActorsIndex, actorIndex, applicationType)
	if err != nil {
		return err
	}

	if applicationType != MorseOrphanedApplication {
		// Add an unstaked actor with the given balance
		_ = unstakedBalance
	}

	mf.morseStateExport.AppState.Application.Applications = append(
		mf.morseStateExport.AppState.Application.Applications,
		morseApplication,
	)

	morseClaimableAccount, err := mf.AddMorseClaimableAccount(morseApplication)
	if err != nil {
		return err
	}

	mf.morseAccountState.Accounts[allActorsIndex] = morseClaimableAccount

	return nil
}

func (mf *MorseMigrationFixtures) addSupplier(
	allActorsIndex,
	actorIndex uint64,
	supplierType MorseSupplierActorType,
) error {
	morseSupplier, unstakedBalance, err := mf.GenMorseSupplier(allActorsIndex, actorIndex, supplierType)
	if err != nil {
		return err
	}

	if supplierType != MorseOrphanedSupplier {
		// Add an unstaked actor with the given balance
		_ = unstakedBalance
	}

	mf.morseStateExport.AppState.Pos.Validators = append(
		mf.morseStateExport.AppState.Pos.Validators,
		morseSupplier,
	)

	morseClaimableAccount, err := mf.AddMorseClaimableAccount(morseSupplier)
	if err != nil {
		return err
	}

	mf.morseAccountState.Accounts[allActorsIndex] = morseClaimableAccount

	return nil
}

// TODO_TECHDEBT: Remove once all tests use the new fixture generation.
// NewMorseStateExportAndAccountStateBytes returns:
//   - A serialized MorseStateExport.
//     This is the JSON output of `pocket util export-genesis-for-reset`.
//     It is used to generate the MorseAccountState.
//   - Its corresponding MorseAccountState.
//     This is the JSON output of `pocketd tx migration collect-morse-accounts`.
//     It is used to persist the canonical Morse migration state (snapshot) from on Shannon.
//
// The states are populated with:
// - Random account addresses
// - Monotonically increasing balances/stakes
// - Unstaked, application, supplier accounts are distributed according to the given distribution function.
func NewMorseStateExportAndAccountStateBytes(
	numAccounts int,
	distributionFn MorseAccountActorTypeDistributionFn,
) (morseStateExportBz []byte, morseAccountStateBz []byte, err error) {
	morseStateExport, morseAccountState, err := NewMorseStateExportAndAccountState(numAccounts, distributionFn)
	if err != nil {
		return nil, nil, err
	}

	morseStateExportBz, err = cmtjson.Marshal(morseStateExport)
	if err != nil {
		return nil, nil, err
	}

	morseAccountStateBz, err = cmtjson.Marshal(morseAccountState)
	if err != nil {
		return nil, nil, err
	}

	return morseStateExportBz, morseAccountStateBz, nil
}

// NewMorseStateExportAndAccountState returns MorseStateExport and MorseAccountState
// structs populated with:
// - Random account addresses
// - Monotonically increasing balances/stakes
// - Unstaked, application, supplier accounts are distributed according to the given distribution function.
func NewMorseStateExportAndAccountState(
	numAccounts int,
	distributionFn MorseAccountActorTypeDistributionFn,
) (export *migrationtypes.MorseStateExport, state *migrationtypes.MorseAccountState, err error) {
	morseStateExport := &migrationtypes.MorseStateExport{
		AppHash: "",
		AppState: &migrationtypes.MorseTendermintAppState{
			Application: &migrationtypes.MorseApplications{},
			Auth:        &migrationtypes.MorseAuth{},
			Pos:         &migrationtypes.MorsePos{},
		},
	}

	morseAccountState := &migrationtypes.MorseAccountState{
		Accounts: make([]*migrationtypes.MorseClaimableAccount, numAccounts),
	}

	for i := 0; i < numAccounts; i++ {
		morseAccountType := distributionFn(uint64(i))
		switch morseAccountType {
		case MorseUnstakedActor:
			// No-op; no staked actor to generate.
		case MorseApplicationActor:
			// Add an application.
			morseStateExport.AppState.Application.Applications = append(
				morseStateExport.AppState.Application.Applications,
				GenMorseApplication(uint64(i)),
			)
		case MorseSupplierActor:
			// Add a supplier.
			// In Morse, a Node (aka a Servicer) is a Shannon Supplier.
			// In Morse, Validators are, by default, the top 1000 staked Nodes.
			morseStateExport.AppState.Pos.Validators = append(
				morseStateExport.AppState.Pos.Validators,
				GenMorseValidator(uint64(i)),
			)
		default:
			panic(fmt.Sprintf("unknown morse account stake state %q", morseAccountType))
		}

		// Add an account (regardless of whether it is staked or not).
		// All MorseClaimableAccount fixtures get an unstaked balance.
		morseAccountJSONBz, err := cmtjson.Marshal(GenMorseAccount(uint64(i)))
		if err != nil {
			return nil, nil, err
		}

		morseStateExport.AppState.Auth.Accounts = append(
			morseStateExport.AppState.Auth.Accounts,
			&migrationtypes.MorseAuthAccount{
				Type:  migrationtypes.MorseExternallyOwnedAccountType,
				Value: morseAccountJSONBz,
			},
		)

		// Add the account to the morseAccountState.
		morseClaimableAccount, err := GenMorseClaimableAccount(uint64(i), distributionFn)
		if err != nil {
			return nil, nil, err
		}

		morseAccountState.Accounts[i] = morseClaimableAccount
	}

	return morseStateExport, morseAccountState, nil
}

// GenMorsePrivateKey creates a new ed25519 private key from the given seed.
func GenMorsePrivateKey(seed uint64) cometcrypto.PrivKey {
	seedBz := make([]byte, 8)
	binary.LittleEndian.PutUint64(seedBz, seed)

	return cometcrypto.GenPrivKeyFromSecret(seedBz)
}

// GenMorseUnstakedBalanceAmount returns an amount by applying the given index to
// a pattern such that the generated amount is unique to the unstaked balance
// (as opposed to actor stake(s)) and the given index.
// E.g.:
// - GenMorseApplicationStakeAmount(0)  =  1000001
// - GenMorseApplicationStakeAmount(1)  =  2000002
// - GenMorseApplicationStakeAmount(10) = 10000010
func GenMorseUnstakedBalanceAmount(index uint64) int64 {
	index++
	return int64(1e6*index + index) // index_000_00index
}

// GenMorseSupplierStakeAmount returns an amount by applying the given index to
// a pattern such that the generated amount is unique to the supplier stake
// (as opposed to the unstaked balance or application stake) and the given index.
// E.g.:
// - GenMorseApplicationStakeAmount(0)  =  10100
// - GenMorseApplicationStakeAmount(1)  =  20200
// - GenMorseApplicationStakeAmount(10) = 101000
func GenMorseSupplierStakeAmount(index uint64) int64 {
	index++
	return int64(1e4*index + (index * 100)) // index0_index00
}

// GenMorseApplicationStakeAmount returns an amount by applying the given index to
// a pattern such that the generated amount is unique to the application stake
// (as opposed to the unstaked balance or supplier stake) and the given index.
// E.g.:
// - GenMorseApplicationStakeAmount(0)  =  100010
// - GenMorseApplicationStakeAmount(1)  =  200020
// - GenMorseApplicationStakeAmount(10) = 1000100
func GenMorseApplicationStakeAmount(index uint64) int64 {
	index++
	return int64(1e5*index + (index * 10)) // index00_0index0
}

// GenMorseAccount returns a new MorseAccount fixture. The given index is used
// to deterministically generate the account's address and unstaked balance.
func GenMorseAccount(index uint64) *migrationtypes.MorseAccount {
	privKey := GenMorsePrivateKey(index)
	pubKey := privKey.PubKey()
	unstakedBalanceAmount := GenMorseUnstakedBalanceAmount(index)
	unstakedBalance := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, unstakedBalanceAmount)

	return &migrationtypes.MorseAccount{
		Address: pubKey.Address(),
		Coins:   cosmostypes.NewCoins(unstakedBalance),
		PubKey: &migrationtypes.MorsePublicKey{
			Value: pubKey.Bytes(),
		},
	}
}

// GenMorseApplication returns a new MorseApplication fixture. The given index is used
// to deterministically generate the application's address and staked tokens.
func GenMorseApplication(idx uint64) *migrationtypes.MorseApplication {
	privKey := GenMorsePrivateKey(idx)
	pubKey := privKey.PubKey()
	stakeAmount := GenMorseApplicationStakeAmount(idx)

	return &migrationtypes.MorseApplication{
		Address:      pubKey.Address(),
		PublicKey:    pubKey.Bytes(),
		Jailed:       false,
		Status:       2,
		StakedTokens: fmt.Sprintf("%d", stakeAmount),
	}
}

func (mf *MorseMigrationFixtures) GenMorseApplication(
	allActorsIndex uint64,
	actorTypeIndex uint64,
	actorType MorseApplicationActorType,
) (*migrationtypes.MorseApplication, *cosmostypes.Coin, error) {
	privKey, err := mf.generateMorsePrivateKey(allActorsIndex)
	if err != nil {
		return nil, nil, err
	}

	pubKey := privKey.PubKey()

	morseApplication := &migrationtypes.MorseApplication{
		Address:   pubKey.Address(),
		PublicKey: pubKey.Bytes(),
		Jailed:    false,
		Status:    2,
	}

	stakedBalance, unstakedBalance := mf.config.ApplicationStakesConfig.GetStakedAndUnstakedBalances(
		allActorsIndex,
		actorTypeIndex,
		actorType,
		morseApplication,
	)
	morseApplication.StakedTokens = fmt.Sprintf("%d", stakedBalance.Amount.Int64())

	return morseApplication, unstakedBalance, nil
}

// GenMorseValidator returns a new MorseValidator fixture. The given index is used
// to deterministically generate the validator's address and staked tokens.
func GenMorseValidator(idx uint64) *migrationtypes.MorseValidator {
	privKey := GenMorsePrivateKey(idx)
	pubKey := privKey.PubKey()
	stakeAmount := GenMorseSupplierStakeAmount(idx)

	return &migrationtypes.MorseValidator{
		Address:      pubKey.Address(),
		PublicKey:    pubKey.Bytes(),
		Jailed:       false,
		Status:       2,
		StakedTokens: fmt.Sprintf("%d", stakeAmount),
	}
}

// TODO_IN_THIS_COMMIT: move & godoc...
func (mf *MorseMigrationFixtures) GenMorseAccount(
	allActorsIndex uint64,
	actorTypeIndex uint64,
	actorType MorseUnstakedActorType,
) (*migrationtypes.MorseAccount, error) {
	privKey, err := mf.generateMorsePrivateKey(allActorsIndex)
	if err != nil {
		return nil, err
	}
	pubKey := privKey.PubKey()
	address := pubKey.Address()

	switch actorType {
	case MorseEOA:
	// No-op; use the default address.
	case MorseModule:
		address = append(address, []byte{0x00}...)
	case MorseInvalidTooShort:
		address = address[:len(address)-1]
	case MorseInvalidTooLong:
		address = append(address, []byte{0x00}...)
	case MorseNonHex:
		invalidBytes := []byte("invalidhex_")
		address = append(invalidBytes, address[len(invalidBytes):]...)
	}

	morseAccount := &migrationtypes.MorseAccount{
		Address: address,
		PubKey: &migrationtypes.MorsePublicKey{
			Value: pubKey.Bytes(),
		},
	}

	balance := mf.config.UnstakedAccountBalancesConfig.GetBalance(
		allActorsIndex,
		actorTypeIndex,
		actorType,
		morseAccount,
	)
	morseAccount.Coins = cosmostypes.NewCoins(*balance)

	return morseAccount, nil
}

func (mf *MorseMigrationFixtures) GenMorseSupplier(
	allActorsIndex uint64,
	actorTypeIndex uint64,
	actorType MorseSupplierActorType,
) (*migrationtypes.MorseValidator, *cosmostypes.Coin, error) {
	privKey, err := mf.generateMorsePrivateKey(allActorsIndex)
	if err != nil {
		return nil, nil, err
	}

	pubKey := privKey.PubKey()
	morseValidator := &migrationtypes.MorseValidator{
		Address:   pubKey.Address(),
		PublicKey: pubKey.Bytes(),
		Jailed:    false,
		Status:    2,
	}

	stakedBalance, unstakedBalance := mf.config.SupplierStakesConfig.GetStakedAndUnstakedBalances(
		allActorsIndex,
		actorTypeIndex,
		actorType,
		morseValidator,
	)
	morseValidator.StakedTokens = fmt.Sprintf("%d", stakedBalance.Amount.Int64())

	return morseValidator, unstakedBalance, nil
}

// TODO_IN_THIS_COMMIT: move & godoc...
func (mf *MorseMigrationFixtures) generateMorsePrivateKey(index uint64) (cometcrypto.PrivKey, error) {
	privKey := GenMorsePrivateKey(index)
	if err := mf.trackPrivateKey(index, privKey); err != nil {
		return nil, err
	}

	return privKey, nil
}

// TODO_IN_THIS_COMMIT: move & godoc...
func (mf *MorseMigrationFixtures) trackPrivateKey(index uint64, privKey cometcrypto.PrivKey) error {
	if _, exists := mf.morseKeysByIndex[index]; exists {
		return fmt.Errorf("duplicate private key for index %d", index)
	}

	address := privKey.PubKey().Address().String()
	if _, exists := mf.morseKeyIndexexByAddr[address]; exists {
		return fmt.Errorf("duplicate private key for address %q", address)
	}

	if _, exists := mf.morseKeysByAddr[address]; exists {
		return fmt.Errorf("duplicate private key for address %q", address)
	}

	mf.morseKeysByIndex[index] = privKey
	mf.morseKeyIndexexByAddr[address] = index
	mf.morseKeysByAddr[address] = privKey

	return nil
}

// GenMorseClaimableAccount returns a new MorseClaimableAccount fixture. The given index is used
// to deterministically generate the account's address and staked tokens. The given distribution
// function is used to determine the account's actor type (and stake if applicable).
func GenMorseClaimableAccount(
	index uint64,
	distributionFn func(uint64) MorseAccountActorType,
) (*migrationtypes.MorseClaimableAccount, error) {
	if distributionFn == nil {
		return nil, fmt.Errorf("distributionFn cannot be nil")
	}

	var appStakeAmount,
		supplierStakeAmount int64
	privKey := GenMorsePrivateKey(index)
	pubKey := privKey.PubKey()

	morseAccountActorType := distributionFn(index)
	switch morseAccountActorType {
	case MorseUnstakedActor:
		// No-op.
	case MorseApplicationActor:
		appStakeAmount = GenMorseApplicationStakeAmount(index)
	case MorseSupplierActor:
		supplierStakeAmount = GenMorseSupplierStakeAmount(index)
	default:
		return nil, fmt.Errorf("unknown morse account stake state %q", morseAccountActorType)
	}

	// All MorseClaimableAccount fixtures get an unstaked balance.
	unstakedBalanceAmount := GenMorseUnstakedBalanceAmount(index)

	return &migrationtypes.MorseClaimableAccount{
		MorseSrcAddress:  pubKey.Address().String(),
		UnstakedBalance:  cosmostypes.NewInt64Coin(volatile.DenomuPOKT, unstakedBalanceAmount),
		SupplierStake:    cosmostypes.NewInt64Coin(volatile.DenomuPOKT, supplierStakeAmount),
		ApplicationStake: cosmostypes.NewInt64Coin(volatile.DenomuPOKT, appStakeAmount),
		// ShannonDestAddress: (intentionally omitted).
		// ClaimedAtHeight:    (intentionally omitted)
	}, nil
}

const (
	invalidMorseAddrTooShortFmt = "FFFFFFFF00%.8x"
	invalidMorseAddrTooLongFmt  = "FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF00%.8x"
	invalidMorseAddrNonHexFmt   = "invalidhex_%x"
)

// GenerateInvalidAddressMorseStateExportAndAccountState generates a MorseStateExport and MorseAccountState
// with invalid addresses for the following cases, and for each actor type (i.e. unstaked, app, suppler):
// - invalid hex
// - too short
// - too long
func GenerateInvalidAddressMorseStateExportAndAccountState(t *testing.T) (*migrationtypes.MorseStateExport, *migrationtypes.MorseAccountState) {
	t.Helper()

	invalidAddrMorseStateExport, invalidAddrMorseAccountState, err := NewMorseStateExportAndAccountState(
		9, NewRoundRobinClusteredAllMorseAccountActorTypes(3))
	require.NoError(t, err)

	for i, morseAuthAccount := range invalidAddrMorseStateExport.AppState.Auth.Accounts {
		// There should be no module accounts.
		require.NotEqual(t, migrationtypes.MorseModuleAccountType, morseAuthAccount.GetType())

		morseAccount, err := morseAuthAccount.AsMorseAccount()
		require.NoError(t, err)

		originalAddr := morseAccount.Address
		switch i % 3 {
		case 0:
			// invalid hex
			morseAccount.Address = []byte(fmt.Sprintf(invalidMorseAddrNonHexFmt, i))
		case 1:
			// too short
			hexAddress, addrErr := hex.DecodeString(fmt.Sprintf(invalidMorseAddrTooShortFmt, i))
			require.NoError(t, addrErr)

			morseAccount.Address = hexAddress
		case 2:
			// too long
			hexAddress, addrErr := hex.DecodeString(fmt.Sprintf(invalidMorseAddrTooLongFmt, i))
			require.NoError(t, addrErr)

			morseAccount.Address = hexAddress
		}

		err = morseAuthAccount.SetAddress(morseAccount.Address)
		require.NoError(t, err)

		// Search for apps or suppliers corresponding to originalAddr and replace the addresses
		for _, app := range invalidAddrMorseStateExport.AppState.Application.Applications {
			if bytes.Equal(app.Address.Bytes(), originalAddr) {
				app.Address = morseAccount.Address
				break
			}
		}

		for _, supplier := range invalidAddrMorseStateExport.AppState.Pos.Validators {
			if bytes.Equal(supplier.Address.Bytes(), originalAddr) {
				supplier.Address = morseAccount.Address
				break
			}
		}

		// Search for MorseClaimableAccounts in MorseAccountState corresponding to originalAddr and replace the addresses
		for _, morseClaimableAccount := range invalidAddrMorseAccountState.Accounts {
			if morseClaimableAccount.GetMorseSrcAddress() == originalAddr.String() {
				morseClaimableAccount.MorseSrcAddress = morseAccount.Address.String()
				break
			}
		}
	}

	return invalidAddrMorseStateExport, invalidAddrMorseAccountState
}

// GenerateModuleAddressMorseStateExportAndAccountState generates a MorseStateExport and MorseAccountState
// with the following module addresses, and ONLY as a liquid account:
// - dao
// - fee_collector
// - application_stake_tokens_pool
// - staked_tokens_pool
func GenerateModuleAddressMorseStateExportAndAccountState(t *testing.T, moduleAccountNames []string) (*migrationtypes.MorseStateExport, *migrationtypes.MorseAccountState) {
	t.Helper()

	moduleAddrMorseStateExport, moduleAddrMorseAccountState, err := NewMorseStateExportAndAccountState(
		len(moduleAccountNames), NewRoundRobinClusteredAllMorseAccountActorTypes(3))
	require.NoError(t, err)

	for i, morseAuthAccount := range moduleAddrMorseStateExport.AppState.Auth.Accounts {
		require.NotEqual(t, migrationtypes.MorseModuleAccountType, morseAuthAccount.GetType())

		morseModuleAccountName := moduleAccountNames[i]

		// Omit stake pool module accounts from the MorseAccountState.
		shouldIncludeInMorseAccountState := true
		for _, skippedModuleAccountName := range migrationtypes.MorseStakePoolModuleAccountNames {
			if morseModuleAccountName == skippedModuleAccountName {
				shouldIncludeInMorseAccountState = false
				break
			}
		}
		if !shouldIncludeInMorseAccountState {
			continue
		}

		// Promote MorseAccounts to MorseModuleAccounts in MorseStateExport.
		morseAccount, err := morseAuthAccount.AsMorseAccount()
		require.NoError(t, err)

		morseModuleAccount := &migrationtypes.MorseModuleAccount{
			Name:        morseModuleAccountName,
			BaseAccount: *morseAccount,
		}

		morseModuleAccountJSONBz, err := cmtjson.Marshal(morseModuleAccount)
		require.NoError(t, err)

		morseAuthAccount.Type = migrationtypes.MorseModuleAccountType
		morseAuthAccount.Value = morseModuleAccountJSONBz

		// Update MorseAccountState to hold module account names for each MorseClaimableAccount's address.
		moduleAddrMorseAccountState.Accounts[i].MorseSrcAddress = moduleAccountNames[i]
	}

	return moduleAddrMorseStateExport, moduleAddrMorseAccountState
}
