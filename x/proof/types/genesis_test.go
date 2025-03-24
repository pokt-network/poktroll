package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/pocket/testutil/sample"
	"github.com/pokt-network/pocket/x/proof/types"
	sessiontypes "github.com/pokt-network/pocket/x/session/types"
)

func TestGenesisState_Validate(t *testing.T) {
	randSupplierOperatorAddr := sample.AccAddress()
	mockSessionId := "mock_session_id"

	tests := []struct {
		desc     string
		genState *types.GenesisState
		isValid  bool
	}{
		{
			desc:     "default is valid",
			genState: types.DefaultGenesis(),
			isValid:  true,
		},
		{
			desc: "valid genesis state",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ClaimList: []types.Claim{
					{
						SupplierOperatorAddress: sample.AccAddress(),
						SessionHeader: &sessiontypes.SessionHeader{
							SessionId:          mockSessionId,
							ApplicationAddress: sample.AccAddress(),
						},
						RootHash: []byte{1, 2, 3},
					},
				},
				// TODO_TEST: finish genesis proof list validation.
				//ProofList: []types.Proof{
				//	{
				//		SupplierOperatorAddress:    sample.AccAddress(),
				//		SessionHeader:      &sessiontypes.SessionHeader{
				//			SessionId:          mockSessionId,
				//			ApplicationAddress: sample.AccAddress(),
				//		},
				//		ClosestMerkleProof: validMerkleProof,
				//	},
				//},
				// this line is used by starport scaffolding # types/genesis/validField
			},
			isValid: true,
		},
		{
			desc: "duplicated claim",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ClaimList: []types.Claim{
					{
						SupplierOperatorAddress: randSupplierOperatorAddr,
						SessionHeader: &sessiontypes.SessionHeader{
							SessionId:          mockSessionId,
							ApplicationAddress: sample.AccAddress(),
						},
						RootHash: []byte{1, 2, 3},
					},
					{
						SupplierOperatorAddress: randSupplierOperatorAddr,
						SessionHeader: &sessiontypes.SessionHeader{
							SessionId:          mockSessionId,
							ApplicationAddress: sample.AccAddress(),
						},
						RootHash: []byte{1, 2, 3},
					},
				},
			},
			isValid: false,
		},
		{
			desc: "empty root hash",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ClaimList: []types.Claim{
					{
						SupplierOperatorAddress: sample.AccAddress(),
						SessionHeader: &sessiontypes.SessionHeader{
							SessionId:          mockSessionId,
							ApplicationAddress: sample.AccAddress(),
						},
						RootHash: []byte{},
					},
				},
			},
			isValid: false,
		},
		{
			desc: "nil root hash",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ClaimList: []types.Claim{
					{
						SupplierOperatorAddress: sample.AccAddress(),
						SessionHeader: &sessiontypes.SessionHeader{
							SessionId:          mockSessionId,
							ApplicationAddress: sample.AccAddress(),
						},
						RootHash: nil,
					},
				},
			},
			isValid: false,
		},
		// TODO_TEST:: finish genesis proof list validation.
		//{
		//	desc: "duplicated proof",
		//	genState: &types.GenesisState{
		//		ProofList: []types.Proof{
		//			{
		//				Index: "0",
		//			},
		//			{
		//				Index: "0",
		//			},
		//		},
		//	},
		//	valid: false,
		//},
		// this line is used by starport scaffolding # types/genesis/testcase
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := test.genState.Validate()
			if test.isValid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
