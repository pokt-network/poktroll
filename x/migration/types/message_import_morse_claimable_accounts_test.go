package types

import (
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
)

func TestMsgImportMorseClaimableAccounts_ValidateBasic(t *testing.T) {
	validMsg, err := NewMsgImportMorseClaimableAccounts(sample.AccAddress(), MorseAccountState{})
	require.NoError(t, err)

	invalidMsg := *validMsg
	invalidMsg.MorseAccountStateHash = []byte("invalid_hash")

	tests := []struct {
		name string
		msg  MsgImportMorseClaimableAccounts
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgImportMorseClaimableAccounts{
				Authority: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "invalid morse account state hash",
			msg:  invalidMsg,
			err:  ErrMorseAccountsImport,
		},
		{
			name: "valid address",
			msg:  *validMsg,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}
