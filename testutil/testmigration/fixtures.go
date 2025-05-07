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
		morseStateExport.AppState.Auth.Accounts = append(
			morseStateExport.AppState.Auth.Accounts,
			&migrationtypes.MorseAuthAccount{
				Type: migrationtypes.MorseExternallyOwnedAccountType,
				Value: &migrationtypes.MorseAuthAccount_MorseAccount{
					MorseAccount: GenMorseAccount(uint64(i)),
				},
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

		morseAccount := morseAuthAccount.GetMorseAccount()
		originalAddr := morseAccount.Address
		switch i % 3 {
		case 0:
			// invalid hex
			morseAuthAccount.SetAddress([]byte(fmt.Sprintf(invalidMorseAddrNonHexFmt, i)))
		case 1:
			// too short
			hexAddress, err := hex.DecodeString(fmt.Sprintf(invalidMorseAddrTooShortFmt, i))
			require.NoError(t, err)

			morseAuthAccount.SetAddress(hexAddress)
		case 2:
			// too long
			hexAddress, err := hex.DecodeString(fmt.Sprintf(invalidMorseAddrTooLongFmt, i))
			require.NoError(t, err)

			morseAuthAccount.SetAddress(hexAddress)
		}

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
		morseAuthAccount.Type = migrationtypes.MorseModuleAccountType
		morseAuthAccount.Value = &migrationtypes.MorseAuthAccount_MorseModuleAccount{
			MorseModuleAccount: &migrationtypes.MorseModuleAccount{
				Name:         morseModuleAccountName,
				MorseAccount: *morseAuthAccount.GetMorseAccount(),
			},
		}

		// Update MorseAccountState to hold module account names for each MorseClaimableAccount's address.
		moduleAddrMorseAccountState.Accounts[i].MorseSrcAddress = moduleAccountNames[i]
	}

	return moduleAddrMorseStateExport, moduleAddrMorseAccountState
}
