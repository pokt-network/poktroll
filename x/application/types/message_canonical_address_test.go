package types

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
)

func TestApplicationMsgs_RejectNonCanonicalAddresses(t *testing.T) {
	app, gateway := sample.AccAddressBech32(), sample.AccAddressBech32()
	upperApp, upperGateway := strings.ToUpper(app), strings.ToUpper(gateway)

	for desc, msg := range map[string]interface{ ValidateBasic() error }{
		"stake":                &MsgStakeApplication{Address: upperApp},
		"unstake":              &MsgUnstakeApplication{Address: upperApp},
		"delegate app":         &MsgDelegateToGateway{AppAddress: upperApp, GatewayAddress: gateway},
		"delegate gateway":     &MsgDelegateToGateway{AppAddress: app, GatewayAddress: upperGateway},
		"undelegate app":       &MsgUndelegateFromGateway{AppAddress: upperApp, GatewayAddress: gateway},
		"undelegate gateway":   &MsgUndelegateFromGateway{AppAddress: app, GatewayAddress: upperGateway},
		"transfer source":      &MsgTransferApplication{SourceAddress: upperApp, DestinationAddress: gateway},
		"transfer destination": &MsgTransferApplication{SourceAddress: gateway, DestinationAddress: upperApp},
	} {
		t.Run(desc, func(t *testing.T) {
			err := msg.ValidateBasic()
			require.Error(t, err)
			require.Contains(t, err.Error(), "canonical")
		})
	}
	require.NoError(t, (&MsgDelegateToGateway{AppAddress: app, GatewayAddress: gateway}).ValidateBasic())
}
