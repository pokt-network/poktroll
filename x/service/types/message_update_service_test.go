package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestMsgUpdateService_ValidateBasic(t *testing.T) {
	validAddr := sample.AccAddressBech32()
	otherAddr := sample.AccAddressBech32()

	tests := []struct {
		name string
		msg  MsgUpdateService
		err  error
	}{
		{
			name: "valid - successful update service",
			msg: MsgUpdateService{
				OwnerAddress: validAddr,
				Service: sharedtypes.Service{
					Id:                   "svc1",
					Name:                 "test service",
					ComputeUnitsPerRelay: 1,
					OwnerAddress:         validAddr,
				},
			},
		},
		{
			name: "invalid - invalid owner address",
			msg: MsgUpdateService{
				OwnerAddress: "invalid_address",
				Service: sharedtypes.Service{
					Id:                   "svc1",
					Name:                 "test service",
					ComputeUnitsPerRelay: 1,
					OwnerAddress:         validAddr,
				},
			},
			err: ErrServiceInvalidAddress,
		},
		{
			name: "invalid - owner address mismatch",
			msg: MsgUpdateService{
				OwnerAddress: validAddr,
				Service: sharedtypes.Service{
					Id:                   "svc1",
					Name:                 "test service",
					ComputeUnitsPerRelay: 1,
					OwnerAddress:         otherAddr, // different address
				},
			},
			err: ErrServiceInvalidOwnerAddress,
		},
		{
			name: "invalid - empty service id",
			msg: MsgUpdateService{
				OwnerAddress: validAddr,
				Service: sharedtypes.Service{
					Id:                   "", // empty ID
					Name:                 "test service",
					ComputeUnitsPerRelay: 1,
					OwnerAddress:         validAddr,
				},
			},
			err: sharedtypes.ErrSharedInvalidServiceId,
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

func TestNewMsgUpdateService(t *testing.T) {
	validAddr := sample.AccAddressBech32()
	service := sharedtypes.Service{
		Id:                   "svc1",
		Name:                 "test service",
		ComputeUnitsPerRelay: 1,
		OwnerAddress:         validAddr,
	}

	msg := NewMsgUpdateService(validAddr, service)

	require.Equal(t, validAddr, msg.OwnerAddress)
	require.Equal(t, service, msg.Service)
}