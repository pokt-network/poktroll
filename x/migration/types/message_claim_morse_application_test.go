package types_test

import (
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/pocket/testutil/sample"
	migrationtypes "github.com/pokt-network/pocket/x/migration/types"
	sharedtypes "github.com/pokt-network/pocket/x/shared/types"
)

const testServiceId = "svc1"

func TestMsgClaimMorseApplication_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  migrationtypes.MsgClaimMorseApplication
		err  error
	}{
		{
			name: "invalid ShannonDestAddress",
			msg: migrationtypes.MsgClaimMorseApplication{
				ShannonDestAddress: "invalid_address",
				MorseSrcAddress:    sample.MorseAddressHex(),
				MorseSignature:     mockMorseSignature,
				ServiceConfig: &sharedtypes.ApplicationServiceConfig{
					ServiceId: testServiceId,
				},
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "invalid MorseSrcAddress",
			msg: migrationtypes.MsgClaimMorseApplication{
				ShannonDestAddress: sample.AccAddress(),
				MorseSrcAddress:    "invalid_address",
				MorseSignature:     mockMorseSignature,
				ServiceConfig: &sharedtypes.ApplicationServiceConfig{
					ServiceId: testServiceId,
				},
			},
			err: migrationtypes.ErrMorseApplicationClaim,
		}, {
			name: "invalid service ID (empty)",
			msg: migrationtypes.MsgClaimMorseApplication{
				ShannonDestAddress: sample.AccAddress(),
				MorseSrcAddress:    sample.MorseAddressHex(),
				MorseSignature:     mockMorseSignature,
				ServiceConfig: &sharedtypes.ApplicationServiceConfig{
					ServiceId: "",
				},
			},
			err: migrationtypes.ErrMorseApplicationClaim,
		}, {
			name: "invalid service ID (too long)",
			msg: migrationtypes.MsgClaimMorseApplication{
				ShannonDestAddress: sample.AccAddress(),
				MorseSrcAddress:    sample.MorseAddressHex(),
				MorseSignature:     mockMorseSignature,
				ServiceConfig: &sharedtypes.ApplicationServiceConfig{
					ServiceId: "xxxxxxxxxxxxxxxxxxxx",
				},
			},
			err: migrationtypes.ErrMorseApplicationClaim,
		}, {
			name: "invalid empty MorseSignature",
			msg: migrationtypes.MsgClaimMorseApplication{
				ShannonDestAddress: sample.AccAddress(),
				MorseSrcAddress:    sample.MorseAddressHex(),
				MorseSignature:     nil,
				ServiceConfig: &sharedtypes.ApplicationServiceConfig{
					ServiceId: testServiceId,
				},
			},
			err: migrationtypes.ErrMorseApplicationClaim,
		}, {
			name: "valid claim message",
			msg: migrationtypes.MsgClaimMorseApplication{
				ShannonDestAddress: sample.AccAddress(),
				MorseSrcAddress:    sample.MorseAddressHex(),
				MorseSignature:     mockMorseSignature,
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
