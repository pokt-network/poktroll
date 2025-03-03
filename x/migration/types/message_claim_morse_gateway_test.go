package types_test

import (
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

func TestMsgClaimMorseGateway_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  migrationtypes.MsgClaimMorseGateway
		err  error
	}{
		{
			name: "invalid ShannonDestAddress",
			msg: migrationtypes.MsgClaimMorseGateway{
				ShannonDestAddress: "invalid_address",
				MorseSrcAddress:    sample.MorseAddressHex(),
				MorseSignature:     mockMorseSignature,
				Stake:              suppliertypes.DefaultMinStake,
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "invalid MorseSrcAddress",
			msg: migrationtypes.MsgClaimMorseGateway{
				ShannonDestAddress: sample.AccAddress(),
				MorseSrcAddress:    "invalid_address",
				MorseSignature:     mockMorseSignature,
				Stake:              suppliertypes.DefaultMinStake,
			},
			err: migrationtypes.ErrMorseGatewayClaim,
		}, {
			name: "invalid empty MorseSignature",
			msg: migrationtypes.MsgClaimMorseGateway{
				ShannonDestAddress: sample.AccAddress(),
				MorseSrcAddress:    sample.MorseAddressHex(),
				MorseSignature:     nil,
				Stake:              gatewaytypes.DefaultMinStake,
			},
			err: migrationtypes.ErrMorseGatewayClaim,
		}, {
			name: "valid claim message",
			msg: migrationtypes.MsgClaimMorseGateway{
				ShannonDestAddress: sample.AccAddress(),
				MorseSrcAddress:    sample.MorseAddressHex(),
				MorseSignature:     mockMorseSignature,
				Stake:              suppliertypes.DefaultMinStake,
			},
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
