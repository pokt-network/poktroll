package testmigration

import (
	"encoding/binary"
	"fmt"

	cometcrypto "github.com/cometbft/cometbft/crypto/ed25519"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/regen-network/gocuke"
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

// MorseAccountActorTypeDistributionFn is a function which returns a MorseAccountActorType
// derived from the given index. It is intended to be used in conjunction with
// MorseStateExport and MorseAccountState fixture generation logic.
type MorseAccountActorTypeDistributionFn func(index uint64) MorseAccountActorType

// RoundRobinAllMorseAccountActorTypes cyclically returns each MorseAccountActorType, one after the other, as the index increases.
// It is used to map a test account index to the test actor that's generated.
func RoundRobinAllMorseAccountActorTypes(index uint64) MorseAccountActorType {
	return MorseAccountActorType(index % uint64(NumMorseAccountActorTypes))
}

// AllUnstakedMorseAccountActorType returns MorseUnstakedActor for every index.
func AllUnstakedMorseAccountActorType(index uint64) MorseAccountActorType {
	return NewSingleMorseAccountActorTypeFn(MorseUnstakedActor)(index)
}

// NewSingleMorseAccountActorTypeFn returns a MorseAccountActorTypeDistributionFn
// which returns the given actor type for every index.
func NewSingleMorseAccountActorTypeFn(actorType MorseAccountActorType) MorseAccountActorTypeDistributionFn {
	return func(_ uint64) MorseAccountActorType {
		return actorType
	}
}

// NewMorseStateExportAndAccountStateBytes returns:
//   - A serialized MorseStateExport.
//     This is the JSON output of `pocket util export-genesis-for-reset`.
//     It is used to generate the MorseAccountState.
//   - Its corresponding MorseAccountState.
//     This is the JSON output of `poktrolld migrate collect-morse-accounts`.
//     It is used to persist the canonical Morse migration state (snapshot) from on Shannon.
//
// The states are populated with:
// - Random account addresses
// - Monotonically increasing balances/stakes
// - Unstaked, application, supplier accounts are distributed according to the given distribution function.
func NewMorseStateExportAndAccountStateBytes(
	t gocuke.TestingT,
	numAccounts int,
	distributionFn MorseAccountActorTypeDistributionFn,
) (morseStateExportBz []byte, morseAccountStateBz []byte) {
	morseStateExport, morseAccountState := NewMorseStateExportAndAccountState(t, numAccounts, distributionFn)

	var err error
	morseStateExportBz, err = cmtjson.Marshal(morseStateExport)
	require.NoError(t, err)

	morseAccountStateBz, err = cmtjson.Marshal(morseAccountState)
	require.NoError(t, err)

	return morseStateExportBz, morseAccountStateBz
}

// NewMorseStateExportAndAccountState returns MorseStateExport and MorseAccountState
// structs populated with:
// - Random account addresses
// - Monotonically increasing balances/stakes
// - Unstaked, application, supplier accounts are distributed according to the given distribution function.
func NewMorseStateExportAndAccountState(
	t gocuke.TestingT,
	numAccounts int,
	distributionFn MorseAccountActorTypeDistributionFn,
) (export *migrationtypes.MorseStateExport, state *migrationtypes.MorseAccountState) {
	t.Helper()

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
				GenMorseApplication(t, uint64(i)),
			)
		case MorseSupplierActor:
			// Add a supplier.
			// In Morse, a Node (aka a Servicer) is a Shannon Supplier.
			// In Morse, Validators are, by default, the top 1000 staked Nodes.
			morseStateExport.AppState.Pos.Validators = append(
				morseStateExport.AppState.Pos.Validators,
				GenMorseValidator(t, uint64(i)),
			)
		default:
			panic(fmt.Sprintf("unknown morse account stake state %q", morseAccountType))
		}

		// Add an account (regardless of whether it is staked or not).
		// All MorseClaimableAccount fixtures get an unstaked balance.
		morseStateExport.AppState.Auth.Accounts = append(
			morseStateExport.AppState.Auth.Accounts,
			&migrationtypes.MorseAuthAccount{
				Type:  "posmint/Account",
				Value: GenMorseAccount(t, uint64(i)),
			},
		)

		// Add the account to the morseAccountState.
		morseAccountState.Accounts[i] = GenMorseClaimableAccount(t, uint64(i), distributionFn)
	}

	return morseStateExport, morseAccountState
}

// GenMorsePrivateKey creates a new ed25519 private key from the given seed.
func GenMorsePrivateKey(t gocuke.TestingT, seed uint64) cometcrypto.PrivKey {
	t.Helper()

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
func GenMorseAccount(t gocuke.TestingT, index uint64) *migrationtypes.MorseAccount {
	privKey := GenMorsePrivateKey(t, index)
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
func GenMorseApplication(t gocuke.TestingT, idx uint64) *migrationtypes.MorseApplication {
	privKey := GenMorsePrivateKey(t, idx)
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

// GenMorseValidator returns a new MorseValidator fixture. The given index is used
// to deterministically generate the validator's address and staked tokens.
func GenMorseValidator(t gocuke.TestingT, idx uint64) *migrationtypes.MorseValidator {
	privKey := GenMorsePrivateKey(t, idx)
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

// GenMorseClaimableAccount returns a new MorseClaimableAccount fixture. The given index is used
// to deterministically generate the account's address and staked tokens. The given distribution
// function is used to determine the account's actor type (and stake if applicable).
func GenMorseClaimableAccount(
	t gocuke.TestingT,
	index uint64,
	distributionFn func(uint64) MorseAccountActorType,
) *migrationtypes.MorseClaimableAccount {
	require.NotNil(t, distributionFn)

	var appStakeAmount,
		supplierStakeAmount int64
	privKey := GenMorsePrivateKey(t, index)
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
		t.Fatalf("unknown morse account stake state %q", morseAccountActorType)
	}

	// All MorseClaimableAccount fixtures get an unstaked balance.
	unstakedBalanceAmount := GenMorseUnstakedBalanceAmount(index)

	return &migrationtypes.MorseClaimableAccount{
		MorseSrcAddress:  pubKey.Address().String(),
		PublicKey:        pubKey.Bytes(),
		UnstakedBalance:  cosmostypes.NewInt64Coin(volatile.DenomuPOKT, unstakedBalanceAmount),
		SupplierStake:    cosmostypes.NewInt64Coin(volatile.DenomuPOKT, supplierStakeAmount),
		ApplicationStake: cosmostypes.NewInt64Coin(volatile.DenomuPOKT, appStakeAmount),
		// ShannonDestAddress: (intentionally omitted).
		// ClaimedAtHeight:    (intentionally omitted)
	}
}
