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
	session *sessiontypes.Session,
	supplierAddress string,
) *relayer.MinedRelay {
	t.Helper()

	relay := servicetypes.Relay{
		Req: &servicetypes.RelayRequest{
			Meta: servicetypes.RelayRequestMetadata{
				SessionHeader:   session.Header,
				Signature:       []byte("request_signature"),
				SupplierAddress: supplierAddress,
			},
			Payload: []byte("request_payload"),
		},
		Res: &servicetypes.RelayResponse{
			Meta: servicetypes.RelayResponseMetadata{
				SessionHeader:     session.Header,
				SupplierSignature: []byte("supplier_signature"),
			},
			Payload: []byte("response_payload"),
		},
	}

	// TODO_TECHDEBT(@red-0ne, #446): Centralize the configuration for the SMT spec.
	// TODO_TECHDEBT(@red-0ne): marshal using canonical codec.
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
