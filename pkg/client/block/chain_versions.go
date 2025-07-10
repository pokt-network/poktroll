package block

import "github.com/hashicorp/go-version"

// TODO(v0.1.26): Remove this once all actors (validators, gateways, relayminers)
// on the network are upgraded to v0.1.26.
//
// chainVersionAddPayloadHashInRelayResponse is the version of the chain that:
// - Introduced the payload hash in RelayResponse.
// - Removed the full payload from RelayResponse.
var chainVersionAddPayloadHashInRelayResponse *version.Version

func init() {
	var err error
	if chainVersionAddPayloadHashInRelayResponse, err = version.NewVersion("v0.1.25"); err != nil {
		panic("failed to parse chain version v0.1.25: " + err.Error())
	}
}

// TODO(v0.1.26): Remove this once all actors (validators, gateways, relayminers)
// on the network are upgraded to v0.1.26.
//
// Compare the chain version with the chainVersionAddPayloadHashInRelayResponse.
//
// - If chainVersion >= chainVersionAddPayloadHashInRelayResponse:
//   - Compute the payload hash
//   - Include the payload hash in the RelayResponse.
//   - Omit the full payload from the RelayResponse.
//
// - If chainVersion < chainVersionAddPayloadHashInRelayResponse:
//   - Remain backward compatible with older versions of the Network
//   - Include the full payload in the RelayResponse.
//   - Do not compute the payload hash at all
func IsChainAfterAddPayloadHashInRelayResponse(chainVersion *version.Version) bool {
	return chainVersion.GreaterThanOrEqual(chainVersionAddPayloadHashInRelayResponse)
}
