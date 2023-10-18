package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"pocket/testutil/sample"
	sharedtypes "pocket/x/shared/types"
	"pocket/x/supplier/types"
)

func TestGenesisState_Validate(t *testing.T) {
	addr1 := sample.AccAddress()
	stake1 := sdk.NewCoin("upokt", sdk.NewInt(100))

	addr2 := sample.AccAddress()
	stake2 := sdk.NewCoin("upokt", sdk.NewInt(100))

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

				SupplierList: []sharedtypes.Supplier{
					{
						Address: addr1,
						Stake:   &stake1,
					},
					{
						Address: addr2,
						Stake:   &stake2,
					},
				},
				// this line is used by starport scaffolding # types/genesis/validField
			},
			valid: true,
		},
		{
			desc: "invalid - zero supplier stake",
			genState: &types.GenesisState{
				SupplierList: []sharedtypes.Supplier{
					{
						Address: addr1,
						Stake:   &stake1,
					},
					{
						Address: addr2,
						Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(0)},
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - negative supplier stake",
			genState: &types.GenesisState{
				SupplierList: []sharedtypes.Supplier{
					{
						Address: addr1,
						Stake:   &stake1,
					},
					{
						Address: addr2,
						Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(-100)},
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - wrong stake denom",
			genState: &types.GenesisState{
				SupplierList: []sharedtypes.Supplier{
					{
						Address: addr1,
						Stake:   &stake1,
					},
					{
						Address: addr2,
						Stake:   &sdk.Coin{Denom: "invalid", Amount: sdk.NewInt(100)},
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - missing denom",
			genState: &types.GenesisState{
				SupplierList: []sharedtypes.Supplier{
					{
						Address: addr1,
						Stake:   &stake1,
					},
					{
						Address: addr2,
						Stake:   &sdk.Coin{Denom: "", Amount: sdk.NewInt(100)},
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - due to duplicated supplier address",
			genState: &types.GenesisState{
				SupplierList: []sharedtypes.Supplier{
					{
						Address: addr1,
						Stake:   &stake1,
					},
					{
						Address: addr1,
						Stake:   &stake2,
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - due to nil supplier stake",
			genState: &types.GenesisState{
				SupplierList: []sharedtypes.Supplier{
					{
						Address: addr1,
						Stake:   &stake1,
					},
					{
						Address: addr2,
						Stake:   nil,
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid - due to missing supplier stake",
			genState: &types.GenesisState{
				SupplierList: []sharedtypes.Supplier{
					{
						Address: addr1,
						Stake:   &stake1,
					},
					{
						Address: addr2,
						// Explicitly missing stake
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
