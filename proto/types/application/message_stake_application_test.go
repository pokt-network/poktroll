package application

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/proto/types/shared"
	"github.com/pokt-network/poktroll/testutil/sample"
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
				Services: []*shared.ApplicationServiceConfig{
					{Service: &shared.Service{Id: "svc1"}},
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
				Services: []*shared.ApplicationServiceConfig{
					{Service: &shared.Service{Id: "svc1"}},
				},
			},
			expectedErr: ErrAppInvalidStake,
		}, {
			desc: "valid address - valid stake",
			msg: MsgStakeApplication{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100)},
				Services: []*shared.ApplicationServiceConfig{
					{Service: &shared.Service{Id: "svc1"}},
				},
			},
		}, {
			desc: "valid address - zero stake",
			msg: MsgStakeApplication{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(0)},
				Services: []*shared.ApplicationServiceConfig{
					{Service: &shared.Service{Id: "svc1"}},
				},
			},
			expectedErr: ErrAppInvalidStake,
		}, {
			desc: "valid address - negative stake",
			msg: MsgStakeApplication{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(-100)},
				Services: []*shared.ApplicationServiceConfig{
					{Service: &shared.Service{Id: "svc1"}},
				},
			},
			expectedErr: ErrAppInvalidStake,
		}, {
			desc: "valid address - invalid stake denom",
			msg: MsgStakeApplication{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "invalid", Amount: math.NewInt(100)},
				Services: []*shared.ApplicationServiceConfig{
					{Service: &shared.Service{Id: "svc1"}},
				},
			},
			expectedErr: ErrAppInvalidStake,
		}, {
			desc: "valid address - invalid stake missing denom",
			msg: MsgStakeApplication{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "", Amount: math.NewInt(100)},
				Services: []*shared.ApplicationServiceConfig{
					{Service: &shared.Service{Id: "svc1"}},
				},
			},
			expectedErr: ErrAppInvalidStake,
		},

		// service related tests
		{
			desc: "valid service configs - multiple services",
			msg: MsgStakeApplication{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100)},
				Services: []*shared.ApplicationServiceConfig{
					{Service: &shared.Service{Id: "svc1"}},
					{Service: &shared.Service{Id: "svc2"}},
				},
			},
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
				Services: []*shared.ApplicationServiceConfig{},
			},
			expectedErr: ErrAppInvalidServiceConfigs,
		},
		{
			desc: "invalid service configs - invalid service ID that's too long",
			msg: MsgStakeApplication{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100)},
				Services: []*shared.ApplicationServiceConfig{
					{Service: &shared.Service{Id: "TooLongId1234567890"}},
				},
			},
			expectedErr: ErrAppInvalidServiceConfigs,
		},
		{
			desc: "invalid service configs - invalid service Name that's too long",
			msg: MsgStakeApplication{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100)},
				Services: []*shared.ApplicationServiceConfig{
					{Service: &shared.Service{
						Id:   "123",
						Name: "abcdefghijklmnopqrstuvwxyzab-abcdefghijklmnopqrstuvwxyzab",
					}},
				},
			},
			expectedErr: ErrAppInvalidServiceConfigs,
		},
		{
			desc: "invalid service configs - invalid service ID that contains invalid characters",
			msg: MsgStakeApplication{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100)},
				Services: []*shared.ApplicationServiceConfig{
					{Service: &shared.Service{Id: "12 45 !"}},
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
