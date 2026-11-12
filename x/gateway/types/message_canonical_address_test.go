package types

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
)

func TestGatewayMsgs_RejectNonCanonicalAddresses(t *testing.T) {
	upper := strings.ToUpper(sample.AccAddressBech32())

	for desc, msg := range map[string]interface{ ValidateBasic() error }{
		"stake":           &MsgStakeGateway{Address: upper},
		"unstake":         &MsgUnstakeGateway{Address: upper},
		"update metadata": &MsgUpdateGatewayMetadata{Address: upper},
	} {
		t.Run(desc, func(t *testing.T) {
			err := msg.ValidateBasic()
			require.ErrorIs(t, err, ErrGatewayInvalidAddress)
			require.Contains(t, err.Error(), "canonical")
		})
	}
}
