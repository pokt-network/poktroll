package rings

import (
	"context"
	"encoding/binary"
	"testing"

	ring "github.com/pokt-network/ring-go"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"

	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/testutil/sample"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

// A relay request ring signature must declare at least two members.
func TestVerifyRelayRequestSignature_RequiresTwoMemberRing(t *testing.T) {
	curve := ring.Secp256k1()
	priv := curve.NewRandomScalar()
	keyRing, err := ring.NewKeyRing(curve, 2, priv, 0)
	require.NoError(t, err)
	msg := sha3.Sum256([]byte("m"))
	sig, err := keyRing.Sign(msg, priv)
	require.NoError(t, err)
	sigBz, err := sig.Serialize()
	require.NoError(t, err)

	undersizedSigBz := make([]byte, len(sigBz))
	copy(undersizedSigBz, sigBz)
	binary.BigEndian.PutUint32(undersizedSigBz[:4], 0)

	rc := &ringClient{logger: polyzero.NewLogger()}
	relayRequest := &servicetypes.RelayRequest{
		Meta: servicetypes.RelayRequestMetadata{
			SessionHeader: &sessiontypes.SessionHeader{
				ApplicationAddress:      sample.AccAddressBech32(),
				SessionId:               "session",
				ServiceId:               "svc",
				SessionStartBlockHeight: 1,
				SessionEndBlockHeight:   2,
			},
			Signature: undersizedSigBz,
		},
	}

	err = rc.VerifyRelayRequestSignature(context.Background(), relayRequest)
	require.ErrorContains(t, err, "at least 2")
}
