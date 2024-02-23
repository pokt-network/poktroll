package types

import (
	"testing"

	"github.com/pokt-network/poktroll/testutil/sample"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"

	"github.com/stretchr/testify/require"
)

func TestMsgCreateClaim_ValidateBasic(t *testing.T) {
	tests := []struct {
		desc string

		msg         MsgCreateClaim
		expectedErr error
	}{
		{
			desc: "invalid address",

			msg: MsgCreateClaim{
				SupplierAddress: "invalid_address",
			},
			expectedErr: ErrProofInvalidAddress,
		},
		{
			desc: "valid address but invalid session start height",

			msg: MsgCreateClaim{
				SupplierAddress: sample.AccAddress(),
				SessionHeader: &sessiontypes.SessionHeader{
					SessionStartBlockHeight: -1, // Invalid start height
				},
			},
			expectedErr: ErrProofInvalidSessionStartHeight,
		},
		{
			desc: "valid address and session start height but invalid session ID",

			msg: MsgCreateClaim{
				SupplierAddress: sample.AccAddress(),
				SessionHeader: &sessiontypes.SessionHeader{
					SessionStartBlockHeight: 100,
					SessionId:               "", // Invalid session ID
				},
			},
			expectedErr: ErrProofInvalidSessionId,
		},
		{
			desc: "valid address, session start height, session ID but invalid service",

			msg: MsgCreateClaim{
				SupplierAddress: sample.AccAddress(),
				SessionHeader: &sessiontypes.SessionHeader{
					SessionStartBlockHeight: 100,
					SessionId:               "valid_session_id",
					Service: &sharedtypes.Service{
						Id: "invalid_service_id", // Assuming this ID is invalid
					}, // Should trigger error
				},
			},
			expectedErr: ErrProofInvalidService,
		},
		{
			desc: "valid address, session start height, session ID, service but invalid root hash",

			msg: MsgCreateClaim{
				SupplierAddress: sample.AccAddress(),
				SessionHeader: &sessiontypes.SessionHeader{
					SessionStartBlockHeight: 100,
					SessionId:               "valid_session_id",
					Service: &sharedtypes.Service{
						Id: "svcId", // Assuming this ID is valid
					},
				},
				RootHash: []byte(""), // Invalid root hash
			},
			expectedErr: ErrProofInvalidClaimRootHash,
		},
		{
			desc: "all valid inputs",

			msg: MsgCreateClaim{
				SupplierAddress: sample.AccAddress(),
				SessionHeader: &sessiontypes.SessionHeader{
					SessionStartBlockHeight: 100,
					SessionId:               "valid_session_id",
					Service: &sharedtypes.Service{
						Id: "svcId", // Assuming this ID is valid
					},
				},
				RootHash: []byte("valid_root_hash"), // Assuming this is valid
			},
			expectedErr: nil,
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := test.msg.ValidateBasic()
			if test.expectedErr != nil {
				require.ErrorIs(t, err, test.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
