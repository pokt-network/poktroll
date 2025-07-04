package block

import "github.com/hashicorp/go-version"

// TODO(v0.1.26): Remove this after the entire network is on v0.1.26.
//
// ChainVersionAddPayloadHashInRelayResponse is the version of the chain that:
// - Introduced the payload hash in RelayResponse.
// - Removed the full payload from RelayResponse.
var ChainVersionAddPayloadHashInRelayResponse *version.Version

func init() {
	var err error
	if ChainVersionAddPayloadHashInRelayResponse, err = version.NewVersion("v0.1.25"); err != nil {
		panic("failed to parse chain version add payload hash in relay response: " + err.Error())
	}
}

// TODO(v0.1.26): Remove this check once the chain version is updated.
//
// Compare the chain version with the signingPayloadHashVersion.
//
// - If chainVersion >= ChainVersionAddPayloadHashInRelayResponse:
//   - Compute the payload hash
//   - Include it in the RelayResponse.
//
// - If chainVersion < ChainVersionAddPayloadHashInRelayResponse:
//   - Remain backward compatible with older versions of the Network
//   - Include the full payload in the RelayResponse.
//   - Do not compute the PayloadHash at all
func IsChainAfterAddPayloadHashInRelayResponse(chainVersion *version.Version) bool {
	return chainVersion.GreaterThanOrEqual(ChainVersionAddPayloadHashInRelayResponse)
}
