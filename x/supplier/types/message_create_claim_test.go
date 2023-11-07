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

		msg MsgCreateClaim
		err error
	}{
		{
			desc: "invalid address",

			msg: MsgCreateClaim{
				SupplierAddress: "invalid_address",
			},
			err: ErrSupplierInvalidAddress,
		},
		{
			desc: "valid address but invalid session start height",

			msg: MsgCreateClaim{
				SupplierAddress: sample.AccAddress(),
				SessionHeader: &sessiontypes.SessionHeader{
					SessionStartBlockHeight: 0, // Invalid start height
				},
			},
			err: ErrSupplierInvalidSessionStartHeight,
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
			err: ErrSupplierInvalidSessionId,
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
			err: ErrSupplierInvalidService,
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
			err: ErrSupplierInvalidClaimRootHash,
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
			err: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
