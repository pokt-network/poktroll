//go:build e2e

package e2e

import (
	"encoding/base64"
	"fmt"

	cometcli "github.com/cometbft/cometbft/libs/cli"
	"github.com/stretchr/testify/require"
)

// TheGatewaySetsItsCardTo sets a staked gateway's card via MsgUpdateGatewayMetadata.
//
// The card is passed with --card-base64 so the CLI's client-side schema validation runs
// (the chain itself only enforces the size limit & never parses the card).
func (s *suite) TheGatewaySetsItsCardTo(gatewayAccName, cardJSON string) {
	s.Helper()

	args := []string{
		"tx", "gateway", "update-gateway-metadata",
		"--card-base64", base64.StdEncoding.EncodeToString([]byte(cardJSON)),
		"--from", gatewayAccName,
		keyRingFlag,
		chainIdFlag,
		"--yes",
	}

	s.broadcastTxAndRequireCommitted(args...)
}

// TheGatewayCardShouldContain asserts that the decoded card of a gateway contains the
// given substring, exercising the `pocketd query gateway card` decode path.
func (s *suite) TheGatewayCardShouldContain(gatewayAccName, expectedSubstr string) {
	s.Helper()

	args := []string{
		"query", "gateway", "card",
		s.getKeyAddress(gatewayAccName),
		fmt.Sprintf("--%s=json", cometcli.OutputFlag),
	}
	res, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries, args...)
	require.NoErrorf(s, err, "failed to query the card of gateway %s", gatewayAccName)

	require.Containsf(s, res.Stdout, expectedSubstr,
		"expected the card of gateway %s to contain %q, got: %s", gatewayAccName, expectedSubstr, res.Stdout)
}
