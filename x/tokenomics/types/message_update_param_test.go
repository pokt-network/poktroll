package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/pocket/testutil/sample"
)

func TestMsgUpdateParam_ValidateBasic(t *testing.T) {
	validMintAllocationPercentages := MintAllocationPercentages{
		Dao:         0.1,
		Proposer:    0.1,
		Supplier:    0.1,
		SourceOwner: 0.1,
		Application: 0.6,
	}

	tests := []struct {
		name string
		msg  MsgUpdateParam

		expectedErr error
	}{
		{
			name: "invalid: authority address invalid",
			msg: MsgUpdateParam{
				Authority: "invalid_address",
				Name:      "", // Doesn't matter for this test
				AsType: &MsgUpdateParam_AsMintAllocationPercentages{
					AsMintAllocationPercentages: &validMintAllocationPercentages,
				},
			},

			expectedErr: ErrTokenomicsAddressInvalid,
		},
		{
			name: "invalid: param name incorrect (non-existent)",
			msg: MsgUpdateParam{
				Authority: sample.AccAddress(),
				Name:      "nonexistent",
				AsType: &MsgUpdateParam_AsString{
					AsString: sample.AccAddress(),
				},
			},
			expectedErr: ErrTokenomicsParamNameInvalid,
		},
		{
			name: "invalid: invalid mint allocation percentages",
			msg: MsgUpdateParam{
				Authority: sample.AccAddress(),
				Name:      "mint_allocation_percentages",
				AsType: &MsgUpdateParam_AsMintAllocationPercentages{
					AsMintAllocationPercentages: &MintAllocationPercentages{
						Dao:         0,
						Proposer:    0,
						Supplier:    0,
						SourceOwner: 0,
						Application: 0,
					},
				},
			},
			expectedErr: ErrTokenomicsParamInvalid,
		},
		{
			name: "invalid: global inflation per claim less than 0",
			msg: MsgUpdateParam{
				Authority: sample.AccAddress(),
				Name:      ParamGlobalInflationPerClaim,
				AsType: &MsgUpdateParam_AsFloat{
					AsFloat: -0.1,
				},
			},
			expectedErr: ErrTokenomicsParamInvalid,
		},
		{
			name: "valid: correct address, param name, and type",
			msg: MsgUpdateParam{
				Authority: sample.AccAddress(),
				Name:      ParamMintAllocationPercentages,
				AsType: &MsgUpdateParam_AsMintAllocationPercentages{
					AsMintAllocationPercentages: &validMintAllocationPercentages,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.expectedErr != nil {
				require.ErrorContains(t, err, tt.expectedErr.Error())
				return
			}
			require.NoError(t, err)
		})
	}
}
