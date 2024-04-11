package testrelayer

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/relayer"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

// NewMinedRelay returns a new mined relay with the given session start and end
// heights on the session header, and the bytes and hash fields populated.
func NewMinedRelay(
	t *testing.T,
	sessionStartHeight int64,
	sessionEndHeight int64,
) *relayer.MinedRelay {
	relay := servicetypes.Relay{
		Req: &servicetypes.RelayRequest{
			Meta: servicetypes.RelayRequestMetadata{
				SessionHeader: &sessiontypes.SessionHeader{
					SessionStartBlockHeight: sessionStartHeight,
					SessionEndBlockHeight:   sessionEndHeight,
				},
			},
		},
		Res: &servicetypes.RelayResponse{},
	}

	// TODO_TECHDEBT(#446): Centralize the configuration for the SMT spec.
	// TODO_BLOCKER: marshal using canonical codec.
	relayBz, err := relay.Marshal()
	require.NoError(t, err)
	relayHashArr := servicetypes.GetHashFromBytes(relayBz)
	relayHash := relayHashArr[:]

	return &relayer.MinedRelay{
		Relay: relay,
		Bytes: relayBz,
		Hash:  relayHash,
	}
}
