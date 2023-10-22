package types_test

import (
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"pocket/testutil/sample"
	"pocket/x/application/types"
)

func TestGenesisState_Validate(t *testing.T) {
	addr1 := sample.AccAddress()
	stake1 := sdk.NewCoin("upokt", sdk.NewInt(100))

	addr2 := sample.AccAddress()
	stake2 := sdk.NewCoin("upokt", sdk.NewInt(100))

	emptyDelegatees := make([]codectypes.Any, 0)
	pubKey1 := sample.AccPubKey()
	pubKey2 := sample.AccPubKey()
	anyPubKey1, err := codectypes.NewAnyWithValue(pubKey1)
	require.NoError(t, err)
	anyPubKey2, err := codectypes.NewAnyWithValue(pubKey2)
	require.NoError(t, err)
	invalidPubKey, err := codectypes.NewAnyWithValue(&types.Application{})
	require.NoError(t, err)

	tests := []struct {
		desc     string
		genState *types.GenesisState
		valid    bool
	}{
		{
			desc:     "default is valid",
			genState: types.DefaultGenesis(),
			valid:    true,
		},
		{
			desc: "valid genesis state",
			genState: &types.GenesisState{
				ApplicationList: []types.Application{
					{
						Address:          addr1,
						Stake:            &stake1,
						DelegateePubKeys: []codectypes.Any{*anyPubKey1, *anyPubKey2},
					},
					{
						Address:          addr2,
						Stake:            &stake2,
						DelegateePubKeys: []codectypes.Any{*anyPubKey2, *anyPubKey1},
					},
				},
				// this line is used by starport scaffolding # types/genesis/validField
			},
			valid: true,
		},
		{
			desc: "invalid - zero app stake",
			genState: &types.GenesisState{
				ApplicationList: []types.Application{
					{
						Address:          addr1,
						Stake:            &stake1,
						DelegateePubKeys: emptyDelegatees,
					},
					{
						Address:          addr2,
						Stake:            &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(0)},
						DelegateePubKeys: emptyDelegatees,
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - negative application stake",
			genState: &types.GenesisState{
				ApplicationList: []types.Application{
					{
						Address:          addr1,
						Stake:            &stake1,
						DelegateePubKeys: emptyDelegatees,
					},
					{
						Address:          addr2,
						Stake:            &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(-100)},
						DelegateePubKeys: emptyDelegatees,
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - wrong stake denom",
			genState: &types.GenesisState{
				ApplicationList: []types.Application{
					{
						Address:          addr1,
						Stake:            &stake1,
						DelegateePubKeys: emptyDelegatees,
					},
					{
						Address:          addr2,
						Stake:            &sdk.Coin{Denom: "invalid", Amount: sdk.NewInt(100)},
						DelegateePubKeys: emptyDelegatees,
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - missing denom",
			genState: &types.GenesisState{
				ApplicationList: []types.Application{
					{
						Address:          addr1,
						Stake:            &stake1,
						DelegateePubKeys: emptyDelegatees,
					},
					{
						Address:          addr2,
						Stake:            &sdk.Coin{Denom: "", Amount: sdk.NewInt(100)},
						DelegateePubKeys: emptyDelegatees,
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - due to duplicated app address",
			genState: &types.GenesisState{
				ApplicationList: []types.Application{
					{
						Address:          addr1,
						Stake:            &stake1,
						DelegateePubKeys: emptyDelegatees,
					},
					{
						Address:          addr1,
						Stake:            &stake2,
						DelegateePubKeys: emptyDelegatees,
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - due to nil app stake",
			genState: &types.GenesisState{
				ApplicationList: []types.Application{
					{
						Address:          addr1,
						Stake:            &stake1,
						DelegateePubKeys: emptyDelegatees,
					},
					{
						Address:          addr2,
						Stake:            nil,
						DelegateePubKeys: emptyDelegatees,
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - due to missing app stake",
			genState: &types.GenesisState{
				ApplicationList: []types.Application{
					{
						Address:          addr1,
						Stake:            &stake1,
						DelegateePubKeys: emptyDelegatees,
					},
					{
						Address: addr2,
						// Explicitly missing stake
						DelegateePubKeys: emptyDelegatees,
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - due to invalid delegatee pub key",
			genState: &types.GenesisState{
				ApplicationList: []types.Application{
					{
						Address:          addr1,
						Stake:            &stake1,
						DelegateePubKeys: emptyDelegatees,
					},
					{
						Address:          addr2,
						Stake:            &stake2,
						DelegateePubKeys: []codectypes.Any{*invalidPubKey},
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - due to invalid delegatee pub keys",
			genState: &types.GenesisState{
				ApplicationList: []types.Application{
					{
						Address:          addr1,
						Stake:            &stake1,
						DelegateePubKeys: []codectypes.Any{*anyPubKey1},
					},
					{
						Address:          addr2,
						Stake:            &stake2,
						DelegateePubKeys: []codectypes.Any{*invalidPubKey, *anyPubKey2},
					},
				},
			},
			valid: false,
		},
		// this line is used by starport scaffolding # types/genesis/testcase
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.genState.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
