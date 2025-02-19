package types

import (
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const testServiceId = "svc1"

func TestMsgClaimMorseApplication_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgClaimMorseApplication
		err  error
	}{
		{
			name: "invalid ShannonDestAddress",
			msg: MsgClaimMorseApplication{
				ShannonDestAddress: "invalid_address",
				MorseSrcAddress:    sample.MorseAddressHex(),
				MorseSignature:     "mock_signature",
				Stake:              &apptypes.DefaultMinStake,
				ServiceConfig: &sharedtypes.ApplicationServiceConfig{
					ServiceId: testServiceId,
				},
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "invalid MorseSrcAddress",
			msg: MsgClaimMorseApplication{
				ShannonDestAddress: sample.AccAddress(),
				MorseSrcAddress:    "invalid_address",
				MorseSignature:     "mock_signature",
				Stake:              &apptypes.DefaultMinStake,
				ServiceConfig: &sharedtypes.ApplicationServiceConfig{
					ServiceId: testServiceId,
				},
			},
			err: ErrMorseApplicationClaim,
		}, {
			name: "invalid service ID (empty)",
			msg: MsgClaimMorseApplication{
				ShannonDestAddress: sample.AccAddress(),
				MorseSrcAddress:    sample.MorseAddressHex(),
				MorseSignature:     "mock_signature",
				Stake:              &apptypes.DefaultMinStake,
				ServiceConfig: &sharedtypes.ApplicationServiceConfig{
					ServiceId: "",
				},
			},
			err: ErrMorseApplicationClaim,
		}, {
			name: "invalid service ID (too long)",
			msg: MsgClaimMorseApplication{
				ShannonDestAddress: sample.AccAddress(),
				MorseSrcAddress:    sample.MorseAddressHex(),
				MorseSignature:     "mock_signature",
				Stake:              &apptypes.DefaultMinStake,
				ServiceConfig: &sharedtypes.ApplicationServiceConfig{
					ServiceId: "xxxxxxxxxxxxxxxxxxxx",
				},
			},
			err: ErrMorseApplicationClaim,
		}, {
			name: "valid nil stake",
			msg: MsgClaimMorseApplication{
				ShannonDestAddress: sample.AccAddress(),
				MorseSrcAddress:    sample.MorseAddressHex(),
				MorseSignature:     "mock_signature",
				Stake:              nil,
				ServiceConfig: &sharedtypes.ApplicationServiceConfig{
					ServiceId: testServiceId,
				},
			},
		}, {
			name: "valid claim message",
			msg: MsgClaimMorseApplication{
				ShannonDestAddress: sample.AccAddress(),
				MorseSrcAddress:    sample.MorseAddressHex(),
				MorseSignature:     "mock_signature",
				Stake:              &apptypes.DefaultMinStake,
				ServiceConfig: &sharedtypes.ApplicationServiceConfig{
					ServiceId: testServiceId,
				},
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
