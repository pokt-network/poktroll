package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// TODO_CLEANUP: This test has a lot of copy-pasted code from test to test.
// It can be simplified by splitting it into smaller tests where the common
// fields don't need to be explicitly specified from test to test.
func TestMsgStakeSupplier_ValidateBasic(t *testing.T) {
	defaultServicesList := []*sharedtypes.SupplierServiceConfig{
		{
			Service: &sharedtypes.Service{
				Id: "svcId1",
			},
			Endpoints: []*sharedtypes.SupplierEndpoint{
				{
					Url:     "http://localhost:8081",
					RpcType: sharedtypes.RPCType_JSON_RPC,
					Configs: make([]*sharedtypes.ConfigOption, 0),
				},
			},
		},
	}

	tests := []struct {
		name string
		msg  MsgStakeSupplier
		err  error
	}{
		// address related tests
		{
			name: "invalid address - nil stake",
			msg: MsgStakeSupplier{
				Address: "invalid_address",
				// Stake explicitly nil
				Services: defaultServicesList,
			},
			err: ErrSupplierInvalidAddress,
		},

		// stake related tests
		{
			name: "valid address - nil stake",
			msg: MsgStakeSupplier{
				Address: sample.AccAddress(),
				// Stake explicitly nil
				Services: defaultServicesList,
			},
			err: ErrSupplierInvalidStake,
		},
		{
			name: "valid address - valid stake",
			msg: MsgStakeSupplier{
				Address:  sample.AccAddress(),
				Stake:    &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
				Services: defaultServicesList,
			},
		},
		{
			name: "valid address - zero stake",
			msg: MsgStakeSupplier{
				Address:  sample.AccAddress(),
				Stake:    &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(0)},
				Services: defaultServicesList,
			},
			err: ErrSupplierInvalidStake,
		},
		{
			name: "valid address - negative stake",
			msg: MsgStakeSupplier{
				Address:  sample.AccAddress(),
				Stake:    &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(-100)},
				Services: defaultServicesList,
			},
			err: ErrSupplierInvalidStake,
		},
		{
			name: "valid address - invalid stake denom",
			msg: MsgStakeSupplier{
				Address:  sample.AccAddress(),
				Stake:    &sdk.Coin{Denom: "invalid", Amount: sdk.NewInt(100)},
				Services: defaultServicesList,
			},
			err: ErrSupplierInvalidStake,
		},
		{
			name: "valid address - invalid stake missing denom",
			msg: MsgStakeSupplier{
				Address:  sample.AccAddress(),
				Stake:    &sdk.Coin{Denom: "", Amount: sdk.NewInt(100)},
				Services: defaultServicesList,
			},
			err: ErrSupplierInvalidStake,
		},

		// service related tests
		{
			name: "valid service configs - multiple services",
			msg: MsgStakeSupplier{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
				Services: []*sharedtypes.SupplierServiceConfig{
					{
						Service: &sharedtypes.Service{
							Id: "svcId1",
						},
						Endpoints: []*sharedtypes.SupplierEndpoint{
							{
								Url:     "http://localhost:8081",
								RpcType: sharedtypes.RPCType_JSON_RPC,
								Configs: make([]*sharedtypes.ConfigOption, 0),
							},
						},
					},
					{
						Service: &sharedtypes.Service{
							Id: "svcId2",
						},
						Endpoints: []*sharedtypes.SupplierEndpoint{
							{
								Url:     "http://localhost:8082",
								RpcType: sharedtypes.RPCType_GRPC,
								Configs: make([]*sharedtypes.ConfigOption, 0),
							},
						},
					},
				},
			},
		},
		{
			name: "invalid service configs - omitted",
			msg: MsgStakeSupplier{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
				// Services: intentionally omitted
			},
			err: ErrSupplierInvalidServiceConfig,
		},
		{
			name: "invalid service configs - empty",
			msg: MsgStakeSupplier{
				Address:  sample.AccAddress(),
				Stake:    &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
				Services: []*sharedtypes.SupplierServiceConfig{},
			},
			err: ErrSupplierInvalidServiceConfig,
		},
		{
			name: "invalid service configs - invalid service ID that's too long",
			msg: MsgStakeSupplier{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
				Services: []*sharedtypes.SupplierServiceConfig{
					{
						Service: &sharedtypes.Service{
							Id: "123456790",
						},
						Endpoints: []*sharedtypes.SupplierEndpoint{
							{
								Url:     "http://localhost:8080",
								RpcType: sharedtypes.RPCType_JSON_RPC,
								Configs: make([]*sharedtypes.ConfigOption, 0),
							},
						},
					},
				},
			},
			err: ErrSupplierInvalidServiceConfig,
		},
		{
			name: "invalid service configs - invalid service Name that's too long",
			msg: MsgStakeSupplier{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
				Services: []*sharedtypes.SupplierServiceConfig{
					{
						Service: &sharedtypes.Service{
							Id:   "123",
							Name: "abcdefghijklmnopqrstuvwxyzab-abcdefghijklmnopqrstuvwxyzab",
						},
						Endpoints: []*sharedtypes.SupplierEndpoint{
							{
								Url:     "http://localhost:8080",
								RpcType: sharedtypes.RPCType_JSON_RPC,
								Configs: make([]*sharedtypes.ConfigOption, 0),
							},
						},
					},
				},
			},
			err: ErrSupplierInvalidServiceConfig,
		},
		{
			name: "invalid service configs - invalid service ID that contains invalid characters",
			msg: MsgStakeSupplier{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
				Services: []*sharedtypes.SupplierServiceConfig{
					{
						Service: &sharedtypes.Service{
							Id: "12 45 !",
						},
						Endpoints: []*sharedtypes.SupplierEndpoint{
							{
								Url:     "http://localhost:8080",
								RpcType: sharedtypes.RPCType_JSON_RPC,
								Configs: make([]*sharedtypes.ConfigOption, 0),
							},
						},
					},
				},
			},
			err: ErrSupplierInvalidServiceConfig,
		},
		{
			name: "invalid service configs - missing url",
			msg: MsgStakeSupplier{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
				Services: []*sharedtypes.SupplierServiceConfig{
					{
						Service: &sharedtypes.Service{
							Id:   "svcId",
							Name: "name",
						},
						Endpoints: []*sharedtypes.SupplierEndpoint{
							{
								// Url: intentionally omitted
								RpcType: sharedtypes.RPCType_JSON_RPC,
								Configs: make([]*sharedtypes.ConfigOption, 0),
							},
						},
					},
				},
			},
			err: ErrSupplierInvalidServiceConfig,
		},
		{
			name: "invalid service configs - invalid url",
			msg: MsgStakeSupplier{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
				Services: []*sharedtypes.SupplierServiceConfig{
					{
						Service: &sharedtypes.Service{
							Id:   "svcId",
							Name: "name",
						},
						Endpoints: []*sharedtypes.SupplierEndpoint{
							{
								Url:     "I am not a valid URL",
								RpcType: sharedtypes.RPCType_JSON_RPC,
								Configs: make([]*sharedtypes.ConfigOption, 0),
							},
						},
					},
				},
			},
			err: ErrSupplierInvalidServiceConfig,
		},
		{
			name: "invalid service configs - missing rpc type",
			msg: MsgStakeSupplier{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
				Services: []*sharedtypes.SupplierServiceConfig{
					{
						Service: &sharedtypes.Service{
							Id:   "svcId",
							Name: "name",
						},
						Endpoints: []*sharedtypes.SupplierEndpoint{
							{
								Url: "http://localhost:8080",
								// RpcType: intentionally omitted,
								Configs: make([]*sharedtypes.ConfigOption, 0),
							},
						},
					},
				},
			},
			err: ErrSupplierInvalidServiceConfig,
		},
		// TODO_TEST: Need to add more tests around config types
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
