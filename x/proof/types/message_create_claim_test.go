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
					ApplicationAddress:      sample.AccAddressBech32(),
					ServiceId:               "svcId",
					SessionStartBlockHeight: 100,
					SessionEndBlockHeight:   101,
					SessionId:               "valid_session_id",
				},
			},
			expectedErr: types.ErrProofInvalidAddress,
		},
		{
			desc: "invalid session start height",

			msg: types.MsgCreateClaim{
				SupplierOperatorAddress: sample.AccAddressBech32(),
				SessionHeader: &sessiontypes.SessionHeader{
					ApplicationAddress:      sample.AccAddressBech32(),
					ServiceId:               "svcId",
					SessionStartBlockHeight: -1,
					SessionEndBlockHeight:   101,
					SessionId:               "valid_session_id",
				},
			},
			expectedErr: types.ErrProofInvalidSessionHeader,
		},
		{
			desc: "invalid session ID",

			msg: types.MsgCreateClaim{
				SupplierOperatorAddress: sample.AccAddressBech32(),
				SessionHeader: &sessiontypes.SessionHeader{
					ApplicationAddress:      sample.AccAddressBech32(),
					ServiceId:               "svcId",
					SessionStartBlockHeight: 100,
					SessionEndBlockHeight:   101,
					SessionId:               "",
				},
			},
			expectedErr: types.ErrProofInvalidSessionHeader,
		},
		{
			desc: "invalid service",

			msg: types.MsgCreateClaim{
				SupplierOperatorAddress: sample.AccAddressBech32(),
				SessionHeader: &sessiontypes.SessionHeader{
					ApplicationAddress:      sample.AccAddressBech32(),
					ServiceId:               "invalid service id",
					SessionStartBlockHeight: 100,
					SessionEndBlockHeight:   101,
					SessionId:               "valid_session_id",
				},
			},
			expectedErr: types.ErrProofInvalidSessionHeader,
		},
		{
			desc: "invalid 0 length root hash length",

			msg: types.MsgCreateClaim{
				SupplierOperatorAddress: sample.AccAddressBech32(),
				SessionHeader: &sessiontypes.SessionHeader{
					ApplicationAddress:      sample.AccAddressBech32(),
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
			desc: "invalid root hash, too short",
			msg: types.MsgCreateClaim{
				SupplierOperatorAddress: sample.AccAddressBech32(),
				SessionHeader: &sessiontypes.SessionHeader{
					ApplicationAddress:      sample.AccAddressBech32(),
					ServiceId:               "svcId",
					SessionStartBlockHeight: 100,
					SessionEndBlockHeight:   101,
					SessionId:               "valid_session_id",
				},
				RootHash: tooShortSMSTRoot(t), // root hash too short
			},
			expectedErr: types.ErrProofInvalidClaimRootHash,
		},
		{
			desc: "invalid root hash, too long",
			msg: types.MsgCreateClaim{
				SupplierOperatorAddress: sample.AccAddressBech32(),
				SessionHeader: &sessiontypes.SessionHeader{
					ApplicationAddress:      sample.AccAddressBech32(),
					ServiceId:               "svcId",
					SessionStartBlockHeight: 100,
					SessionEndBlockHeight:   101,
					SessionId:               "valid_session_id",
				},
				RootHash: tooLongSMSTRoot(t), // root hash too long
			},
			expectedErr: types.ErrProofInvalidClaimRootHash,
		},
		{
			desc: "invalid root hash, 0 relays count",
			msg: types.MsgCreateClaim{
				SupplierOperatorAddress: sample.AccAddressBech32(),
				SessionHeader: &sessiontypes.SessionHeader{
					ApplicationAddress:      sample.AccAddressBech32(),
					ServiceId:               "svcId",
					SessionStartBlockHeight: 100,
					SessionEndBlockHeight:   101,
					SessionId:               "valid_session_id",
				},
				RootHash: testproof.RandSmstRootWithSumAndCount(t, 1, 0), // Valid root hash but length zero count (invalid content)
			},
			expectedErr: types.ErrProofInvalidClaimRootHash,
		},
		{
			desc: "invalid root hash, zero compute units",
			msg: types.MsgCreateClaim{
				SupplierOperatorAddress: sample.AccAddressBech32(),
				SessionHeader: &sessiontypes.SessionHeader{
					ApplicationAddress:      sample.AccAddressBech32(),
					ServiceId:               "svcId",
					SessionStartBlockHeight: 100,
					SessionEndBlockHeight:   101,
					SessionId:               "valid_session_id",
				},
				RootHash: testproof.RandSmstRootWithSumAndCount(t, 0, 1), // Valid root hash length but zero sum (invalid content)
			},
			expectedErr: types.ErrProofInvalidClaimRootHash,
		},
		{
			desc: "valid create claim message",
			msg: types.MsgCreateClaim{
				SupplierOperatorAddress: sample.AccAddressBech32(),
				SessionHeader: &sessiontypes.SessionHeader{
					ApplicationAddress:      sample.AccAddressBech32(),
					ServiceId:               "svcId",
					SessionStartBlockHeight: 100,
					SessionEndBlockHeight:   101,
					SessionId:               "valid_session_id",
				},
				RootHash: testproof.RandSmstRootWithSumAndCount(t, 1, 1), // Valid root hash
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

// tooShortSMSTRoot returns an invalid SMST root hash with the given sum and count
// by reducing the size of a valid SMST root hash by one byte,
func tooShortSMSTRoot(t *testing.T) []byte {
	return testproof.RandSmstRootWithSumAndCount(t, 1, 1)[:protocol.TrieRootSize-1]
}

// tooLongSMSTRoot returns an invalid SMST root hash with the given sum and count
// by adding an extra byte to a valid SMST root hash.
func tooLongSMSTRoot(t *testing.T) []byte {
	smstRoot := testproof.RandSmstRootWithSumAndCount(t, 1, 1)

	// Append an extra byte to make it longer than the expected size
	longRoot := make([]byte, protocol.TrieRootSize+1)
	copy(longRoot, smstRoot)
	return longRoot
}
