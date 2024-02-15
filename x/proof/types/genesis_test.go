package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

func TestGenesisState_Validate(t *testing.T) {
	randSupplierAddr := sample.AccAddress()
	mockSessionId := "mock_session_id"

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

				ClaimList: []types.Claim{
					{
						SupplierAddress: sample.AccAddress(),
						SessionHeader: &sessiontypes.SessionHeader{
							SessionId:          mockSessionId,
							ApplicationAddress: sample.AccAddress(),
						},
						RootHash: []byte{1, 2, 3},
					},
				},
				ProofList: []types.Proof{
					{
						Index: "0",
					},
					{
						Index: "1",
					},
				},
				// this line is used by starport scaffolding # types/genesis/validField
			},
			valid: true,
		},
		{
			desc: "duplicated claim",
			genState: &types.GenesisState{
				ClaimList: []types.Claim{
					{
						SupplierAddress: randSupplierAddr,
						SessionHeader: &sessiontypes.SessionHeader{
							SessionId:          mockSessionId,
							ApplicationAddress: sample.AccAddress(),
						},
						RootHash: []byte{1, 2, 3},
					},
					{
						SupplierAddress: randSupplierAddr,
						SessionHeader: &sessiontypes.SessionHeader{
							SessionId:          mockSessionId,
							ApplicationAddress: sample.AccAddress(),
						},
						RootHash: []byte{1, 2, 3},
					},
				},
			},
			valid: false,
		},
		{
			desc: "empty root hash",
			genState: &types.GenesisState{
				ClaimList: []types.Claim{
					{
						SupplierAddress: sample.AccAddress(),
						SessionHeader: &sessiontypes.SessionHeader{
							SessionId:          mockSessionId,
							ApplicationAddress: sample.AccAddress(),
						},
						RootHash: []byte{},
					},
				},
			},
			valid: false,
		},
		{
			desc: "nil root hash",
			genState: &types.GenesisState{
				ClaimList: []types.Claim{
					{
						SupplierAddress: sample.AccAddress(),
						SessionHeader: &sessiontypes.SessionHeader{
							SessionId:          mockSessionId,
							ApplicationAddress: sample.AccAddress(),
						},
						RootHash: nil,
					},
				},
			},
			valid: false,
		},
		{
			desc: "duplicated proof",
			genState: &types.GenesisState{
				ProofList: []types.Proof{
					{
						Index: "0",
					},
					{
						Index: "0",
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
