package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestSessionHeader_ValidateBasic(t *testing.T) {
	tests := []struct {
		desc string
		sh   types.SessionHeader
		err  error
	}{
		{
			desc: "invalid - invalid application address",
			sh: types.SessionHeader{
				ApplicationAddress:      "invalid_address",
				SessionId:               "valid_session_id",
				Service:                 &sharedtypes.Service{},
				SessionStartBlockHeight: 100,
				SessionEndBlockHeight:   101,
			},
			err: types.ErrSessionInvalidAppAddress,
		},
		{
			desc: "invalid - empty session id",
			sh: types.SessionHeader{
				ApplicationAddress:      sample.AccAddress(),
				SessionId:               "",
				Service:                 &sharedtypes.Service{},
				SessionStartBlockHeight: 100,
				SessionEndBlockHeight:   101,
			},
			err: types.ErrSessionInvalidSessionId,
		},
		{
			desc: "invalid - nil service",
			sh: types.SessionHeader{
				ApplicationAddress:      sample.AccAddress(),
				SessionId:               "valid_session_id",
				Service:                 nil,
				SessionStartBlockHeight: 100,
				SessionEndBlockHeight:   101,
			},
			err: types.ErrSessionInvalidService,
		},
		{
			desc: "invalid - start block height greater than end block height",
			sh: types.SessionHeader{
				ApplicationAddress:      sample.AccAddress(),
				SessionId:               "valid_session_id",
				Service:                 &sharedtypes.Service{},
				SessionStartBlockHeight: 100,
				SessionEndBlockHeight:   99,
			},
			err: types.ErrSessionInvalidBlockHeight,
		},
		{
			desc: "valid",
			sh: types.SessionHeader{
				ApplicationAddress:      sample.AccAddress(),
				SessionId:               "valid_session_id",
				Service:                 &sharedtypes.Service{},
				SessionStartBlockHeight: 100,
				SessionEndBlockHeight:   101,
			},
			err: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := tt.sh.ValidateBasic()
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
