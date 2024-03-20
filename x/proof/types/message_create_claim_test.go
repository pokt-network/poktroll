package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestMsgCreateClaim_ValidateBasic(t *testing.T) {
	tests := []struct {
		desc string

		msg         MsgCreateClaim
		expectedErr error
	}{
		{
			desc: "invalid supplier address",

			msg: MsgCreateClaim{
				SupplierAddress: "invalid_address",
				SessionHeader: &sessiontypes.SessionHeader{
					ApplicationAddress: sample.AccAddress(),
					Service: &sharedtypes.Service{
						Id: "svcId", // Assuming this ID is valid
					},
					SessionStartBlockHeight: 100,
					SessionEndBlockHeight:   101,
					SessionId:               "valid_session_id",
				},
			},
			expectedErr: ErrProofInvalidAddress,
		},
		{
			desc: "valid supplier address but invalid session start height",

			msg: MsgCreateClaim{
				SupplierAddress: sample.AccAddress(),
				SessionHeader: &sessiontypes.SessionHeader{
					ApplicationAddress: sample.AccAddress(),
					Service: &sharedtypes.Service{
						Id: "svcId", // Assuming this ID is valid
					},
					SessionStartBlockHeight: -1,
					SessionEndBlockHeight:   101,
					SessionId:               "valid_session_id",
				},
			},
			expectedErr: ErrProofInvalidSessionHeader,
		},
		{
			desc: "valid supplier address and session start height but invalid session ID",

			msg: MsgCreateClaim{
				SupplierAddress: sample.AccAddress(),
				SessionHeader: &sessiontypes.SessionHeader{
					ApplicationAddress: sample.AccAddress(),
					Service: &sharedtypes.Service{
						Id: "svcId", // Assuming this ID is valid
					},
					SessionStartBlockHeight: 100,
					SessionEndBlockHeight:   101,
					SessionId:               "",
				},
			},
			expectedErr: ErrProofInvalidSessionHeader,
		},
		{
			desc: "valid address, session start height, session ID but invalid service",

			msg: MsgCreateClaim{
				SupplierAddress: sample.AccAddress(),
				SessionHeader: &sessiontypes.SessionHeader{
					ApplicationAddress: sample.AccAddress(),
					Service: &sharedtypes.Service{
						Id: "invalid service id", // invalid service ID
					},
					SessionStartBlockHeight: 100,
					SessionEndBlockHeight:   101,
					SessionId:               "valid_session_id",
				},
			},
			expectedErr: ErrProofInvalidSessionHeader,
		},
		{
			desc: "valid address, session start height, session ID, service but invalid root hash",

			msg: MsgCreateClaim{
				SupplierAddress: sample.AccAddress(),
				SessionHeader: &sessiontypes.SessionHeader{
					ApplicationAddress: sample.AccAddress(),
					Service: &sharedtypes.Service{
						Id: "svcId", // Assuming this ID is valid
					},
					SessionStartBlockHeight: 100,
					SessionEndBlockHeight:   101,
					SessionId:               "valid_session_id",
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
					ApplicationAddress: sample.AccAddress(),
					Service: &sharedtypes.Service{
						Id: "svcId", // Assuming this ID is valid
					},
					SessionStartBlockHeight: 100,
					SessionEndBlockHeight:   101,
					SessionId:               "valid_session_id",
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
