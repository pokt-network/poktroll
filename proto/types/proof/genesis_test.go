package proof_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/proto/types/proof"
	"github.com/pokt-network/poktroll/proto/types/session"
	"github.com/pokt-network/poktroll/testutil/sample"
)

func TestGenesisState_Validate(t *testing.T) {
	randSupplierAddr := sample.AccAddress()
	mockSessionId := "mock_session_id"

	tests := []struct {
		desc     string
		genState *proof.GenesisState
		isValid  bool
	}{
		{
			desc:     "default is valid",
			genState: proof.DefaultGenesis(),
			isValid:  true,
		},
		{
			desc: "valid genesis state",
			genState: &proof.GenesisState{
				Params: proof.DefaultParams(),
				ClaimList: []proof.Claim{
					{
						SupplierAddress: sample.AccAddress(),
						SessionHeader: &session.SessionHeader{
							SessionId:          mockSessionId,
							ApplicationAddress: sample.AccAddress(),
						},
						RootHash: []byte{1, 2, 3},
					},
				},
				// TODO_TEST: finish genesis proof list validation.
				//ProofList: []types.Proof{
				//	{
				//		SupplierAddress:    sample.AccAddress(),
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
			genState: &proof.GenesisState{
				Params: proof.DefaultParams(),
				ClaimList: []proof.Claim{
					{
						SupplierAddress: randSupplierAddr,
						SessionHeader: &session.SessionHeader{
							SessionId:          mockSessionId,
							ApplicationAddress: sample.AccAddress(),
						},
						RootHash: []byte{1, 2, 3},
					},
					{
						SupplierAddress: randSupplierAddr,
						SessionHeader: &session.SessionHeader{
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
			genState: &proof.GenesisState{
				Params: proof.DefaultParams(),
				ClaimList: []proof.Claim{
					{
						SupplierAddress: sample.AccAddress(),
						SessionHeader: &session.SessionHeader{
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
			genState: &proof.GenesisState{
				Params: proof.DefaultParams(),
				ClaimList: []proof.Claim{
					{
						SupplierAddress: sample.AccAddress(),
						SessionHeader: &session.SessionHeader{
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
