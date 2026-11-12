package types

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
)

func TestServiceMsgs_RejectNonCanonicalAddresses(t *testing.T) {
	owner := sample.AccAddressBech32()
	upper := strings.ToUpper(owner)

	for desc, msg := range map[string]interface{ ValidateBasic() error }{
		"add service owner":      &MsgAddService{OwnerAddress: upper},
		"transfer service owner": &MsgTransferService{OwnerAddress: upper, NewOwnerAddress: owner, ServiceId: "svc"},
		"transfer new owner":     &MsgTransferService{OwnerAddress: owner, NewOwnerAddress: upper, ServiceId: "svc"},
	} {
		t.Run(desc, func(t *testing.T) {
			err := msg.ValidateBasic()
			require.ErrorIs(t, err, ErrServiceInvalidAddress)
			require.Contains(t, err.Error(), "canonical")
		})
	}
}
