package testrelayer

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/miner"
	"github.com/pokt-network/poktroll/x/service/types"
	types2 "github.com/pokt-network/poktroll/x/session/types"
)

// newMinedRelay returns a new mined relay with the given session start and end
// heights on the session header, and the bytes and hash fields populated.
func NewMinedRelay(
	t *testing.T,
	sessionStartHeight int64,
	sessionEndHeight int64,
) *relayer.MinedRelay {
	relay := types.Relay{
		Req: &types.RelayRequest{
			Meta: &types.RelayRequestMetadata{
				SessionHeader: &types2.SessionHeader{
					SessionStartBlockHeight: sessionStartHeight,
					SessionEndBlockHeight:   sessionEndHeight,
				},
			},
		},
		Res: &types.RelayResponse{},
	}

	// TODO_BLOCKER: use canonical codec to serialize the relay
	relayBz, err := relay.Marshal()
	require.NoError(t, err)

	relayHash := HashBytes(t, miner.DefaultRelayHasher, relayBz)

	return &relayer.MinedRelay{
		Relay: relay,
		Bytes: relayBz,
		Hash:  relayHash,
	}
}
