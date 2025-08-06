package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const testServiceId = "svc_id"

func TestSessionHeader_ValidateBasic(t *testing.T) {
	tests := []struct {
		desc          string
		sessionHeader types.SessionHeader
		expectedErr   error
	}{
		{
			desc: "invalid - invalid application address",
			sessionHeader: types.SessionHeader{
				ApplicationAddress:      "invalid_address",
				SessionId:               "valid_session_id",
				ServiceId:               testServiceId,
				SessionStartBlockHeight: 100,
				SessionEndBlockHeight:   101,
			},
			expectedErr: types.ErrSessionInvalidAppAddress,
		},
		{
			desc: "invalid - empty session id",
			sessionHeader: types.SessionHeader{
				ApplicationAddress:      sample.AccAddressBech32(),
				SessionId:               "",
				ServiceId:               testServiceId,
				SessionStartBlockHeight: 100,
				SessionEndBlockHeight:   101,
			},
			expectedErr: types.ErrSessionInvalidSessionId,
		},
		{
			desc: "invalid - empty service id",
			sessionHeader: types.SessionHeader{
				ApplicationAddress:      sample.AccAddressBech32(),
				SessionId:               "valid_session_id",
				ServiceId:               "",
				SessionStartBlockHeight: 100,
				SessionEndBlockHeight:   101,
			},
			expectedErr: sharedtypes.ErrSharedInvalidServiceId,
		},
		{
			desc: "invalid - start block height is 0",
			sessionHeader: types.SessionHeader{
				ApplicationAddress:      sample.AccAddressBech32(),
				SessionId:               "valid_session_id",
				ServiceId:               testServiceId,
				SessionStartBlockHeight: 0,
				SessionEndBlockHeight:   42,
			},
			expectedErr: types.ErrSessionInvalidBlockHeight,
		},
		{
			desc: "invalid - start block height greater than end block height",
			sessionHeader: types.SessionHeader{
				ApplicationAddress:      sample.AccAddressBech32(),
				SessionId:               "valid_session_id",
				ServiceId:               testServiceId,
				SessionStartBlockHeight: 100,
				SessionEndBlockHeight:   99,
			},
			expectedErr: types.ErrSessionInvalidBlockHeight,
		},
		{
			desc: "valid",
			sessionHeader: types.SessionHeader{
				ApplicationAddress:      sample.AccAddressBech32(),
				SessionId:               "valid_session_id",
				ServiceId:               testServiceId,
				SessionStartBlockHeight: 100,
				SessionEndBlockHeight:   101,
			},
			expectedErr: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := test.sessionHeader.ValidateBasic()
			if test.expectedErr != nil {
				require.ErrorContains(t, err, test.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
