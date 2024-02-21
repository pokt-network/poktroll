package types

import (
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/pokt-network/poktroll/testutil/sample"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"

	"github.com/stretchr/testify/require"
)

func TestMsgSubmitProof_ValidateBasic(t *testing.T) {
	testService := &sharedtypes.Service{Id: "svc01"}
	testClosestMerkleProof := []byte{1, 2, 3, 4}

	tests := []struct {
		desc        string
		msg         MsgSubmitProof
		expectedErr error
	}{
		{
			desc: "application bech32 address is invalid",
			msg: MsgSubmitProof{
				SupplierAddress: sample.AccAddress(),
				SessionHeader: &sessiontypes.SessionHeader{
					ApplicationAddress:      "not_a_bech32_address",
					Service:                 testService,
					SessionStartBlockHeight: 0,
					SessionId:               "mock_session_id",
					SessionEndBlockHeight:   5,
				},
				Proof: testClosestMerkleProof,
			},
			expectedErr: sdkerrors.ErrInvalidAddress.Wrapf(
				"application address: %q, error: %s",
				"not_a_bech32_address",
				"decoding bech32 failed: invalid separator index -1",
			),
		},
		{
			desc: "supplier bech32 address is invalid",
			msg: MsgSubmitProof{
				SupplierAddress: "not_a_bech32_address",
				SessionHeader: &sessiontypes.SessionHeader{
					ApplicationAddress:      sample.AccAddress(),
					Service:                 testService,
					SessionStartBlockHeight: 0,
					SessionId:               "mock_session_id",
					SessionEndBlockHeight:   5,
				},
				Proof: testClosestMerkleProof,
			},
			expectedErr: sdkerrors.ErrInvalidAddress.Wrapf(
				"supplier address %q, error: %s",
				"not_a_bech32_address",
				"decoding bech32 failed: invalid separator index -1",
			),
		},
		{
			desc: "session service ID is empty",
			msg: MsgSubmitProof{
				SupplierAddress: sample.AccAddress(),
				SessionHeader: &sessiontypes.SessionHeader{
					ApplicationAddress:      sample.AccAddress(),
					Service:                 &sharedtypes.Service{Id: ""},
					SessionStartBlockHeight: 0,
					SessionId:               "mock_session_id",
					SessionEndBlockHeight:   5,
				},
				Proof: testClosestMerkleProof,
			},
			expectedErr: ErrProofInvalidService.Wrap("proof service ID %q cannot be empty"),
		},
		{
			desc: "valid message metadata",
			msg: MsgSubmitProof{
				SupplierAddress: sample.AccAddress(),
				SessionHeader: &sessiontypes.SessionHeader{
					ApplicationAddress:      sample.AccAddress(),
					Service:                 testService,
					SessionId:               "mock_session_id",
					SessionStartBlockHeight: 0,
					SessionEndBlockHeight:   5,
				},
				Proof: testClosestMerkleProof,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := test.msg.ValidateBasic()
			if test.expectedErr != nil {
				require.ErrorIs(t, err, test.expectedErr)
				require.ErrorContains(t, err, test.expectedErr.Error())
				return
			}
			require.NoError(t, err)
		})
	}
}
