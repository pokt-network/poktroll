package types

import (
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
	testsession "github.com/pokt-network/poktroll/testutil/session"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestMsgSubmitProof_ValidateBasic(t *testing.T) {
	testService := &sharedtypes.Service{Id: "svc01"}
	testClosestMerkleProof := []byte{1, 2, 3, 4}

	tests := []struct {
		desc                           string
		msg                            MsgSubmitProof
		sessionHeaderToExpectedErrorFn func(sessiontypes.SessionHeader) error
	}{
		{
			desc: "application bech32 address is invalid",
			msg: MsgSubmitProof{
				SupplierOperatorAddress: sample.AccAddress(),
				SessionHeader: &sessiontypes.SessionHeader{
					ApplicationAddress:      "not_a_bech32_address",
					Service:                 testService,
					SessionId:               "mock_session_id",
					SessionStartBlockHeight: 1,
					SessionEndBlockHeight:   testsession.GetSessionEndHeightWithDefaultParams(1),
				},
				Proof: testClosestMerkleProof,
			},
			sessionHeaderToExpectedErrorFn: func(sh sessiontypes.SessionHeader) error {
				sessionError := sessiontypes.ErrSessionInvalidAppAddress.Wrapf(
					"%q; (%s)",
					sh.ApplicationAddress,
					"decoding bech32 failed: invalid separator index -1",
				)
				return ErrProofInvalidSessionHeader.Wrapf("%s", sessionError)
			},
		},
		{
			desc: "supplier operator bech32 address is invalid",
			msg: MsgSubmitProof{
				SupplierOperatorAddress: "not_a_bech32_address",
				SessionHeader: &sessiontypes.SessionHeader{
					ApplicationAddress:      sample.AccAddress(),
					Service:                 testService,
					SessionId:               "mock_session_id",
					SessionStartBlockHeight: 1,
					SessionEndBlockHeight:   testsession.GetSessionEndHeightWithDefaultParams(1),
				},
				Proof: testClosestMerkleProof,
			},
			sessionHeaderToExpectedErrorFn: func(sh sessiontypes.SessionHeader) error {
				return sdkerrors.ErrInvalidAddress.Wrapf(
					"supplier operator address %q, error: %s",
					"not_a_bech32_address",
					"decoding bech32 failed: invalid separator index -1",
				)
			},
		},
		{
			desc: "session service ID is empty",
			msg: MsgSubmitProof{
				SupplierOperatorAddress: sample.AccAddress(),
				SessionHeader: &sessiontypes.SessionHeader{
					ApplicationAddress:      sample.AccAddress(),
					Service:                 &sharedtypes.Service{Id: ""},
					SessionId:               "mock_session_id",
					SessionStartBlockHeight: 1,
					SessionEndBlockHeight:   testsession.GetSessionEndHeightWithDefaultParams(1),
				},
				Proof: testClosestMerkleProof,
			},
			sessionHeaderToExpectedErrorFn: func(sh sessiontypes.SessionHeader) error {
				serviceError := sharedtypes.ErrSharedInvalidService.Wrapf("ID: %q", sh.Service.Id)
				sessionError := sessiontypes.ErrSessionInvalidService.Wrapf("%s", serviceError)
				return ErrProofInvalidSessionHeader.Wrapf("%s", sessionError)
			},
		},
		{
			desc: "valid message metadata",
			msg: MsgSubmitProof{
				SupplierOperatorAddress: sample.AccAddress(),
				SessionHeader: &sessiontypes.SessionHeader{
					ApplicationAddress:      sample.AccAddress(),
					Service:                 testService,
					SessionId:               "mock_session_id",
					SessionStartBlockHeight: 1,
					SessionEndBlockHeight:   testsession.GetSessionEndHeightWithDefaultParams(1),
				},
				Proof: testClosestMerkleProof,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := test.msg.ValidateBasic()
			if test.sessionHeaderToExpectedErrorFn != nil {
				expectedErr := test.sessionHeaderToExpectedErrorFn(*test.msg.SessionHeader)
				require.ErrorIs(t, err, expectedErr)
				require.ErrorContains(t, err, expectedErr.Error())
				return
			}
			require.NoError(t, err)
		})
	}
}
