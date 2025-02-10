package testmigration

import (
	"encoding/binary"
	"fmt"
	"math/rand"

	cometcrypto "github.com/cometbft/cometbft/crypto/ed25519"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/regen-network/gocuke"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/volatile"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

// NewMorseStateExportAndAccountStateBytes returns:
//   - A serialized MorseStateExport.
//     This is the JSON output of `pocket util export-genesis-for-reset`.
//     It is used to generate the MorseAccountState.
//   - Its corresponding MorseAccountState.
//     This is the JSON output of `poktrolld migrate collect-morse-accounts`.
//     It is used to persist the canonical Morse migration state from on Shannon.
//
// The states are populated with:
// - Random account addresses
// - Monotonically increasing balances/stakes
// - One application per account
// - One supplier per account
func NewMorseStateExportAndAccountStateBytes(
	t gocuke.TestingT,
	numAccounts int,
) (morseStateExportBz []byte, morseAccountStateBz []byte) {
	morseStateExport, morseAccountState := NewMorseStateExportAndAccountState(t, numAccounts)

	var err error
	morseStateExportBz, err = cmtjson.Marshal(morseStateExport)
	require.NoError(t, err)

	morseAccountStateBz, err = cmtjson.Marshal(morseAccountState)
	require.NoError(t, err)

	return morseStateExportBz, morseAccountStateBz
}

// NewMorseStateExportAndAccountState returns MorseStateExport and MorseAccountState
// structs populated with:
//   - Random account addresses
//   - Monotonically increasing balances/stakes
//   - One application per account
//   - One supplier per account
func NewMorseStateExportAndAccountState(
	t gocuke.TestingT, numAccounts int,
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
		Accounts: make([]*migrationtypes.MorseAccount, numAccounts),
	}

	for i := 1; i < numAccounts+1; i++ {
		seedUint := rand.Uint64()
		seedBz := make([]byte, 8)
		binary.LittleEndian.PutUint64(seedBz, seedUint)
		privKey := cometcrypto.GenPrivKeyFromSecret(seedBz)
		pubKey := privKey.PubKey()
		balanceAmount := int64(1e6*i + i)                                 // i_000_00i
		appStakeAmount := int64(1e5*i + (i * 10))                         //   i00_0i0
		supplierStakeAmount := int64(1e4*i + (i * 100))                   //    i0_i00
		sumAmount := balanceAmount + appStakeAmount + supplierStakeAmount // i_ii0_iii

		// Add an account.
		morseStateExport.AppState.Auth.Accounts = append(
			morseStateExport.AppState.Auth.Accounts,
			&migrationtypes.MorseAuthAccount{
				Type: "posmint/Account",
				Value: &migrationtypes.MorseAccount{
					Address: pubKey.Address(),
					Coins:   cosmostypes.NewCoins(cosmostypes.NewInt64Coin(volatile.DenomuPOKT, balanceAmount)),
					PubKey: &migrationtypes.MorsePublicKey{
						Value: pubKey.Bytes(),
					},
				},
			},
		)

		// Add an application.
		morseStateExport.AppState.Application.Applications = append(
			morseStateExport.AppState.Application.Applications,
			&migrationtypes.MorseApplication{
				Address:      pubKey.Address(),
				PublicKey:    pubKey.Bytes(),
				Jailed:       false,
				Status:       2,
				StakedTokens: fmt.Sprintf("%d", appStakeAmount),
			},
		)

		// Add a supplier.
		morseStateExport.AppState.Pos.Validators = append(
			morseStateExport.AppState.Pos.Validators,
			&migrationtypes.MorseValidator{
				Address:      pubKey.Address(),
				PublicKey:    pubKey.Bytes(),
				Jailed:       false,
				Status:       2,
				StakedTokens: fmt.Sprintf("%d", supplierStakeAmount),
			},
		)

		// Add the account to the morseAccountState.
		morseAccountState.Accounts[i-1] = &migrationtypes.MorseAccount{
			Address: pubKey.Address(),
			Coins:   cosmostypes.NewCoins(cosmostypes.NewInt64Coin(volatile.DenomuPOKT, sumAmount)),
			PubKey: &migrationtypes.MorsePublicKey{
				Value: pubKey.Bytes(),
			},
		}
	}

	return morseStateExport, morseAccountState
}
