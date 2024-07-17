package proof

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/proto/types/session"
	"github.com/pokt-network/poktroll/proto/types/shared"
	"github.com/pokt-network/poktroll/testutil/sample"
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
				SessionHeader: &session.SessionHeader{
					ApplicationAddress:      sample.AccAddress(),
					Service:                 &shared.Service{Id: "svcId"},
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
				SessionHeader: &session.SessionHeader{
					ApplicationAddress:      sample.AccAddress(),
					Service:                 &shared.Service{Id: "svcId"},
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
				SessionHeader: &session.SessionHeader{
					ApplicationAddress:      sample.AccAddress(),
					Service:                 &shared.Service{Id: "svcId"},
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
				SessionHeader: &session.SessionHeader{
					ApplicationAddress:      sample.AccAddress(),
					Service:                 &shared.Service{Id: "invalid service id"},
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
				SessionHeader: &session.SessionHeader{
					ApplicationAddress:      sample.AccAddress(),
					Service:                 &shared.Service{Id: "svcId"},
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
				SessionHeader: &session.SessionHeader{
					ApplicationAddress:      sample.AccAddress(),
					Service:                 &shared.Service{Id: "svcId"},
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
