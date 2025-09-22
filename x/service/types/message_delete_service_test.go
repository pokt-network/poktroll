package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
)

func TestMsgDeleteService_ValidateBasic(t *testing.T) {
	validAddr := sample.AccAddressBech32()

	tests := []struct {
		name string
		msg  MsgDeleteService
		err  error
	}{
		{
			name: "valid - successful delete service",
			msg: MsgDeleteService{
				OwnerAddress: validAddr,
				ServiceId:    "svc1",
			},
		},
		{
			name: "invalid - invalid owner address",
			msg: MsgDeleteService{
				OwnerAddress: "invalid_address",
				ServiceId:    "svc1",
			},
			err: ErrServiceInvalidAddress,
		},
		{
			name: "invalid - empty service id",
			msg: MsgDeleteService{
				OwnerAddress: validAddr,
				ServiceId:    "", // empty service ID
			},
			err: ErrServiceInvalidServiceId,
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

func TestNewMsgDeleteService(t *testing.T) {
	validAddr := sample.AccAddressBech32()
	msg := NewMsgDeleteService(validAddr, "svc1")

	require.Equal(t, validAddr, msg.OwnerAddress)
	require.Equal(t, "svc1", msg.ServiceId)
}