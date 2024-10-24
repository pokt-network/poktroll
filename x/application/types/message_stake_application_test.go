package types

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestMsgStakeApplication_ValidateBasic(t *testing.T) {
	tests := []struct {
		desc        string
		msg         MsgStakeApplication
		expectedErr error
	}{
		// address related tests
		{
			desc: "invalid address - nil stake",
			msg: MsgStakeApplication{
				Address: "invalid_address",
				// Stake explicitly omitted
				Services: []*sharedtypes.ApplicationServiceConfig{
					{ServiceId: "svc1"},
				},
			},
			expectedErr: ErrAppInvalidAddress,
		},

		// stake related tests
		{
			desc: "valid address - nil stake",
			msg: MsgStakeApplication{
				Address: sample.AccAddress(),
				// Stake explicitly omitted
				Services: []*sharedtypes.ApplicationServiceConfig{
					{ServiceId: "svc1"},
				},
			},
			expectedErr: ErrAppInvalidStake,
		}, {
			desc: "valid address - valid stake",
			msg: MsgStakeApplication{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100)},
				Services: []*sharedtypes.ApplicationServiceConfig{
					{ServiceId: "svc1"},
				},
			},
		}, {
			desc: "valid address - zero stake",
			msg: MsgStakeApplication{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(0)},
				Services: []*sharedtypes.ApplicationServiceConfig{
					{ServiceId: "svc1"},
				},
			},
			expectedErr: ErrAppInvalidStake,
		}, {
			desc: "valid address - negative stake",
			msg: MsgStakeApplication{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(-100)},
				Services: []*sharedtypes.ApplicationServiceConfig{
					{ServiceId: "svc1"},
				},
			},
			expectedErr: ErrAppInvalidStake,
		}, {
			desc: "valid address - invalid stake denom",
			msg: MsgStakeApplication{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "invalid", Amount: math.NewInt(100)},
				Services: []*sharedtypes.ApplicationServiceConfig{
					{ServiceId: "svc1"},
				},
			},
			expectedErr: ErrAppInvalidStake,
		}, {
			desc: "valid address - invalid stake missing denom",
			msg: MsgStakeApplication{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "", Amount: math.NewInt(100)},
				Services: []*sharedtypes.ApplicationServiceConfig{
					{ServiceId: "svc1"},
				},
			},
			expectedErr: ErrAppInvalidStake,
		},

		// service related tests
		{
			desc: "invalid service configs - multiple services",
			msg: MsgStakeApplication{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100)},
				Services: []*sharedtypes.ApplicationServiceConfig{
					{ServiceId: "svc1"},
					{ServiceId: "svc2"},
				},
			},
			expectedErr: ErrAppInvalidServiceConfigs,
		},
		{
			desc: "invalid service configs - not present",
			msg: MsgStakeApplication{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100)},
				// Services explicitly omitted
			},
			expectedErr: ErrAppInvalidServiceConfigs,
		},
		{
			desc: "invalid service configs - empty",
			msg: MsgStakeApplication{
				Address:  sample.AccAddress(),
				Stake:    &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100)},
				Services: []*sharedtypes.ApplicationServiceConfig{},
			},
			expectedErr: ErrAppInvalidServiceConfigs,
		},
		{
			desc: "invalid service configs - invalid service ID that's too long",
			msg: MsgStakeApplication{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100)},
				Services: []*sharedtypes.ApplicationServiceConfig{
					{ServiceId: "TooLongId1234567890"},
				},
			},
			expectedErr: ErrAppInvalidServiceConfigs,
		},
		{
			desc: "invalid service configs - invalid service ID that contains invalid characters",
			msg: MsgStakeApplication{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100)},
				Services: []*sharedtypes.ApplicationServiceConfig{
					{ServiceId: "12 45 !"},
				},
			},
			expectedErr: ErrAppInvalidServiceConfigs,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := test.msg.ValidateBasic()
			if test.expectedErr != nil {
				require.ErrorIs(t, err, test.expectedErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
