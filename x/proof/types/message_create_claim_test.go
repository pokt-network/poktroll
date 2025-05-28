package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	testproof "github.com/pokt-network/poktroll/testutil/proof"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

func TestMsgCreateClaim_ValidateBasic(t *testing.T) {
	tests := []struct {
		desc string

		msg         types.MsgCreateClaim
		expectedErr error
	}{
		{
			desc: "invalid supplier operator address",

			msg: types.MsgCreateClaim{
				SupplierOperatorAddress: "invalid_address",
				SessionHeader: &sessiontypes.SessionHeader{
					ApplicationAddress:      sample.AccAddress(),
					ServiceId:               "svcId",
					SessionStartBlockHeight: 100,
					SessionEndBlockHeight:   101,
					SessionId:               "valid_session_id",
				},
			},
			expectedErr: types.ErrProofInvalidAddress,
		},
		{
			desc: "valid supplier operator address but invalid session start height",

			msg: types.MsgCreateClaim{
				SupplierOperatorAddress: sample.AccAddress(),
				SessionHeader: &sessiontypes.SessionHeader{
					ApplicationAddress:      sample.AccAddress(),
					ServiceId:               "svcId",
					SessionStartBlockHeight: -1,
					SessionEndBlockHeight:   101,
					SessionId:               "valid_session_id",
				},
			},
			expectedErr: types.ErrProofInvalidSessionHeader,
		},
		{
			desc: "valid supplier operator address and session start height but invalid session ID",

			msg: types.MsgCreateClaim{
				SupplierOperatorAddress: sample.AccAddress(),
				SessionHeader: &sessiontypes.SessionHeader{
					ApplicationAddress:      sample.AccAddress(),
					ServiceId:               "svcId",
					SessionStartBlockHeight: 100,
					SessionEndBlockHeight:   101,
					SessionId:               "",
				},
			},
			expectedErr: types.ErrProofInvalidSessionHeader,
		},
		{
			desc: "valid operator address, session start height, session ID but invalid service",

			msg: types.MsgCreateClaim{
				SupplierOperatorAddress: sample.AccAddress(),
				SessionHeader: &sessiontypes.SessionHeader{
					ApplicationAddress:      sample.AccAddress(),
					ServiceId:               "invalid service id",
					SessionStartBlockHeight: 100,
					SessionEndBlockHeight:   101,
					SessionId:               "valid_session_id",
				},
			},
			expectedErr: types.ErrProofInvalidSessionHeader,
		},
		{
			desc: "valid operator address, session start height, session ID, service but 0 root hash length",

			msg: types.MsgCreateClaim{
				SupplierOperatorAddress: sample.AccAddress(),
				SessionHeader: &sessiontypes.SessionHeader{
					ApplicationAddress:      sample.AccAddress(),
					ServiceId:               "svcId",
					SessionStartBlockHeight: 100,
					SessionEndBlockHeight:   101,
					SessionId:               "valid_session_id",
				},
				RootHash: []byte(""), // Invalid root hash
			},
			expectedErr: types.ErrProofInvalidClaimRootHash,
		},
		{
			desc: "valid operator address, session start height, session ID, service but root hash too short",
			msg: types.MsgCreateClaim{
				SupplierOperatorAddress: sample.AccAddress(),
				SessionHeader: &sessiontypes.SessionHeader{
					ApplicationAddress:      sample.AccAddress(),
					ServiceId:               "svcId",
					SessionStartBlockHeight: 100,
					SessionEndBlockHeight:   101,
					SessionId:               "valid_session_id",
				},
				RootHash: shortSMSTRoot(), // root hash too short
			},
			expectedErr: types.ErrProofInvalidClaimRootHash,
		},
		{
			desc: "valid operator address, session start height, session ID, service but root hash too long",
			msg: types.MsgCreateClaim{
				SupplierOperatorAddress: sample.AccAddress(),
				SessionHeader: &sessiontypes.SessionHeader{
					ApplicationAddress:      sample.AccAddress(),
					ServiceId:               "svcId",
					SessionStartBlockHeight: 100,
					SessionEndBlockHeight:   101,
					SessionId:               "valid_session_id",
				},
				RootHash: longSMSTRoot(), // root hash too long
			},
			expectedErr: types.ErrProofInvalidClaimRootHash,
		},
		{
			desc: "valid operator address, session start height, session ID, service and valid root hash but with zero count",
			msg: types.MsgCreateClaim{
				SupplierOperatorAddress: sample.AccAddress(),
				SessionHeader: &sessiontypes.SessionHeader{
					ApplicationAddress:      sample.AccAddress(),
					ServiceId:               "svcId",
					SessionStartBlockHeight: 100,
					SessionEndBlockHeight:   101,
					SessionId:               "valid_session_id",
				},
				RootHash: testproof.SmstRootWithSumAndCount(0, 1), // Valid root hash but zero count
			},
			expectedErr: types.ErrProofInvalidClaimRootHash,
		},
		{
			desc: "valid operator address, session start height, session ID, service and valid root hash but with zero sum",
			msg: types.MsgCreateClaim{
				SupplierOperatorAddress: sample.AccAddress(),
				SessionHeader: &sessiontypes.SessionHeader{
					ApplicationAddress:      sample.AccAddress(),
					ServiceId:               "svcId",
					SessionStartBlockHeight: 100,
					SessionEndBlockHeight:   101,
					SessionId:               "valid_session_id",
				},
				RootHash: testproof.SmstRootWithSumAndCount(1, 0), // Valid root hash but zero sum
			},
			expectedErr: types.ErrProofInvalidClaimRootHash,
		},
		{
			desc: "valid root hash",
			msg: types.MsgCreateClaim{
				SupplierOperatorAddress: sample.AccAddress(),
				SessionHeader: &sessiontypes.SessionHeader{
					ApplicationAddress:      sample.AccAddress(),
					ServiceId:               "svcId",
					SessionStartBlockHeight: 100,
					SessionEndBlockHeight:   101,
					SessionId:               "valid_session_id",
				},
				RootHash: testproof.SmstRootWithSumAndCount(1, 1), // Valid root hash
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

// shortSMSTRoot returns a valid SMST root with the given sum and count but
// reduces the size to be one byte shorter than the expected size.
func shortSMSTRoot() []byte {
	return testproof.SmstRootWithSumAndCount(1, 1)[:protocol.TrieRootSumSize-1]
}

// longSMSTRoot returns a valid SMST root with the given sum and count but adds
// an extra byte to make it longer than the expected size.
func longSMSTRoot() []byte {
	smstRoot := testproof.SmstRootWithSumAndCount(1, 1)

	// Append an extra byte to make it longer than the expected size
	longRoot := make([]byte, protocol.TrieRootSumSize+1)
	copy(longRoot, smstRoot)
	return longRoot
}
