package types_test

import (
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/pocket/testutil/sample"
	migrationtypes "github.com/pokt-network/pocket/x/migration/types"
	sharedtypes "github.com/pokt-network/pocket/x/shared/types"
)

var (
	testServices = []*sharedtypes.SupplierServiceConfig{
		{ServiceId: testServiceId},
	}

	testEndpoints = []*sharedtypes.SupplierEndpoint{
		{
			Url:     "http://test.example:1234",
			RpcType: sharedtypes.RPCType_JSON_RPC,
		},
	}

	testRevShare = []*sharedtypes.ServiceRevenueShare{
		{
			Address:            sample.AccAddress(),
			RevSharePercentage: uint64(100),
		},
	}
)

func TestMsgClaimMorseSupplier_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  migrationtypes.MsgClaimMorseSupplier
		err  error
	}{
		{
			name: "invalid ShannonOwnerAddress",
			msg: migrationtypes.MsgClaimMorseSupplier{
				ShannonOwnerAddress: "invalid_address",
				MorseSrcAddress:     sample.MorseAddressHex(),
				MorseSignature:      mockMorseSignature,
				Services:            testServices,
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "invalid ShannonOperatorAddress",
			msg: migrationtypes.MsgClaimMorseSupplier{
				ShannonOwnerAddress:    sample.AccAddress(),
				ShannonOperatorAddress: "invalid_address",
				MorseSrcAddress:        sample.MorseAddressHex(),
				MorseSignature:         mockMorseSignature,
				Services:               testServices,
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "invalid MorseSrcAddress",
			msg: migrationtypes.MsgClaimMorseSupplier{
				ShannonOwnerAddress: sample.AccAddress(),
				MorseSrcAddress:     "invalid_address",
				MorseSignature:      mockMorseSignature,
				Services:            testServices,
			},
			err: migrationtypes.ErrMorseSupplierClaim,
		}, {
			name: "invalid service ID (empty)",
			msg: migrationtypes.MsgClaimMorseSupplier{
				ShannonOwnerAddress: sample.AccAddress(),
				MorseSrcAddress:     sample.MorseAddressHex(),
				MorseSignature:      mockMorseSignature,
				Services: []*sharedtypes.SupplierServiceConfig{
					{ServiceId: ""},
				},
			},
			err: migrationtypes.ErrMorseSupplierClaim,
		}, {
			name: "invalid service ID (too long)",
			msg: migrationtypes.MsgClaimMorseSupplier{
				ShannonOwnerAddress: sample.AccAddress(),
				MorseSrcAddress:     sample.MorseAddressHex(),
				MorseSignature:      mockMorseSignature,
				Services: []*sharedtypes.SupplierServiceConfig{
					{
						ServiceId: "xxxxxxxxxxxxxxxxxxxx",
						Endpoints: testEndpoints,
						RevShare:  testRevShare,
					},
				},
			},
			err: migrationtypes.ErrMorseSupplierClaim,
		}, {
			name: "invalid empty MorseSignature",
			msg: migrationtypes.MsgClaimMorseSupplier{
				ShannonOwnerAddress: sample.AccAddress(),
				MorseSrcAddress:     sample.MorseAddressHex(),
				MorseSignature:      nil,
				Services: []*sharedtypes.SupplierServiceConfig{
					{
						ServiceId: testServiceId,
						Endpoints: testEndpoints,
						RevShare:  testRevShare,
					},
				},
			},
			err: migrationtypes.ErrMorseSupplierClaim,
		}, {
			name: "valid nil stake",
			msg: migrationtypes.MsgClaimMorseSupplier{
				ShannonOwnerAddress: sample.AccAddress(),
				MorseSrcAddress:     sample.MorseAddressHex(),
				MorseSignature:      mockMorseSignature,
				Services: []*sharedtypes.SupplierServiceConfig{
					{
						ServiceId: testServiceId,
						Endpoints: testEndpoints,
						RevShare:  testRevShare,
					},
				},
			},
		}, {
			name: "valid claim message",
			msg: migrationtypes.MsgClaimMorseSupplier{
				ShannonOwnerAddress: sample.AccAddress(),
				MorseSrcAddress:     sample.MorseAddressHex(),
				MorseSignature:      mockMorseSignature,
				Services: []*sharedtypes.SupplierServiceConfig{
					{
						ServiceId: testServiceId,
						Endpoints: testEndpoints,
						RevShare:  testRevShare,
					},
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
