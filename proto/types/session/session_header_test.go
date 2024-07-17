package session_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	types "github.com/pokt-network/poktroll/proto/types/session"
	sharedtypes "github.com/pokt-network/poktroll/proto/types/shared"
	"github.com/pokt-network/poktroll/testutil/sample"
)

func TestSessionHeader_ValidateBasic(t *testing.T) {
	svc := sharedtypes.Service{
		Id:   "svc_id",
		Name: "svc_name",
	}

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
				Service:                 &svc,
				SessionStartBlockHeight: 100,
				SessionEndBlockHeight:   101,
			},
			expectedErr: types.ErrSessionInvalidAppAddress,
		},
		{
			desc: "invalid - empty session id",
			sessionHeader: types.SessionHeader{
				ApplicationAddress:      sample.AccAddress(),
				SessionId:               "",
				Service:                 &svc,
				SessionStartBlockHeight: 100,
				SessionEndBlockHeight:   101,
			},
			expectedErr: types.ErrSessionInvalidSessionId,
		},
		{
			desc: "invalid - nil service",
			sessionHeader: types.SessionHeader{
				ApplicationAddress:      sample.AccAddress(),
				SessionId:               "valid_session_id",
				Service:                 nil,
				SessionStartBlockHeight: 100,
				SessionEndBlockHeight:   101,
			},
			expectedErr: types.ErrSessionInvalidService,
		},
		{
			desc: "invalid - start block height is 0",
			sessionHeader: types.SessionHeader{
				ApplicationAddress:      sample.AccAddress(),
				SessionId:               "valid_session_id",
				Service:                 &svc,
				SessionStartBlockHeight: 0,
				SessionEndBlockHeight:   42,
			},
			expectedErr: types.ErrSessionInvalidBlockHeight,
		},
		{
			desc: "invalid - start block height greater than end block height",
			sessionHeader: types.SessionHeader{
				ApplicationAddress:      sample.AccAddress(),
				SessionId:               "valid_session_id",
				Service:                 &svc,
				SessionStartBlockHeight: 100,
				SessionEndBlockHeight:   99,
			},
			expectedErr: types.ErrSessionInvalidBlockHeight,
		},
		{
			desc: "valid",
			sessionHeader: types.SessionHeader{
				ApplicationAddress:      sample.AccAddress(),
				SessionId:               "valid_session_id",
				Service:                 &svc,
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
				require.ErrorIs(t, err, test.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
