package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"pocket/testutil/sample"
	sharedtypes "pocket/x/shared/types"
)

func TestMsgStakeApplication_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgStakeApplication
		err  error
	}{
		// address tests
		{
			name: "invalid address - nil stake",
			msg: MsgStakeApplication{
				Address: "invalid_address",
				// Stake explicitly nil
				Services: []*sharedtypes.ApplicationServiceConfig{
					{ServiceId: &sharedtypes.ServiceId{Id: "svc1"}},
				},
			},
			err: ErrAppInvalidAddress,
		},

		// stake related tests
		{
			name: "valid address - nil stake",
			msg: MsgStakeApplication{
				Address: sample.AccAddress(),
				// Stake explicitly nil
				Services: []*sharedtypes.ApplicationServiceConfig{
					{ServiceId: &sharedtypes.ServiceId{Id: "svc1"}},
				},
			},
			err: ErrAppInvalidStake,
		}, {
			name: "valid address - valid stake",
			msg: MsgStakeApplication{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
				Services: []*sharedtypes.ApplicationServiceConfig{
					{ServiceId: &sharedtypes.ServiceId{Id: "svc1"}},
				},
			},
		}, {
			name: "valid address - zero stake",
			msg: MsgStakeApplication{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(0)},
				Services: []*sharedtypes.ApplicationServiceConfig{
					{ServiceId: &sharedtypes.ServiceId{Id: "svc1"}},
				},
			},
			err: ErrAppInvalidStake,
		}, {
			name: "valid address - negative stake",
			msg: MsgStakeApplication{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(-100)},
				Services: []*sharedtypes.ApplicationServiceConfig{
					{ServiceId: &sharedtypes.ServiceId{Id: "svc1"}},
				},
			},
			err: ErrAppInvalidStake,
		}, {
			name: "valid address - invalid stake denom",
			msg: MsgStakeApplication{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "invalid", Amount: sdk.NewInt(100)},
				Services: []*sharedtypes.ApplicationServiceConfig{
					{ServiceId: &sharedtypes.ServiceId{Id: "svc1"}},
				},
			},
			err: ErrAppInvalidStake,
		}, {
			name: "valid address - invalid stake missing denom",
			msg: MsgStakeApplication{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "", Amount: sdk.NewInt(100)},
				Services: []*sharedtypes.ApplicationServiceConfig{
					{ServiceId: &sharedtypes.ServiceId{Id: "svc1"}},
				},
			},
			err: ErrAppInvalidStake,
		},

		// service related tests
		{
			name: "invalid service configs - not present",
			msg: MsgStakeApplication{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
				// Services: omitted
			},
			err: ErrAppInvalidServiceConfigs,
		},
		{
			name: "invalid service configs - empty",
			msg: MsgStakeApplication{
				Address:  sample.AccAddress(),
				Stake:    &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
				Services: []*sharedtypes.ApplicationServiceConfig{},
			},
			err: ErrAppInvalidServiceConfigs,
		},
		{
			name: "invalid service configs - invalid service ID that's too long",
			msg: MsgStakeApplication{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
				Services: []*sharedtypes.ApplicationServiceConfig{
					{ServiceId: &sharedtypes.ServiceId{Id: "123456790"}},
				},
			},
			err: ErrAppInvalidServiceConfigs,
		},
		{
			name: "invalid service configs - invalid service Name that's too long",
			msg: MsgStakeApplication{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
				Services: []*sharedtypes.ApplicationServiceConfig{
					{ServiceId: &sharedtypes.ServiceId{
						Id:   "123",
						Name: "abcdefghijklmnopqrstuvwxyzab-abcdefghijklmnopqrstuvwxyzab",
					}},
				},
			},
			err: ErrAppInvalidServiceConfigs,
		},
		{
			name: "invalid service configs - invalid service ID that contains invalid characters",
			msg: MsgStakeApplication{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
				Services: []*sharedtypes.ApplicationServiceConfig{
					{ServiceId: &sharedtypes.ServiceId{Id: "12 45 !"}},
				},
			},
			err: ErrAppInvalidServiceConfigs,
		},
		{
			name: "valid service configs - multiple services",
			msg: MsgStakeApplication{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
				Services: []*sharedtypes.ApplicationServiceConfig{
					{ServiceId: &sharedtypes.ServiceId{Id: "svc1"}},
					{ServiceId: &sharedtypes.ServiceId{Id: "svc2"}},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.err != nil {
				require.ErrorContains(t, err, tt.err.Error())
				return
			}
			require.NoError(t, err)
		})
	}
}
