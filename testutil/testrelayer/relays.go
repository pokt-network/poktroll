package testrelayer

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/rand"

	"github.com/pokt-network/poktroll/pkg/crypto"
	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/pkg/relayer"
	testutilkeyring "github.com/pokt-network/poktroll/testutil/testkeyring"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

// NewUnsignedMinedRelay returns a new mined relay with the given session data,
// as well as the bytes and the hash fields populated.
//
// It DOES NOT populate the signature fields and should only be used in contexts
// where a partial mined relay is enough for testing purposes.
//
// TODO_IMPROVE: It does not (yet) verify against and adhere to the actual
// relay mining difficulty of the service at hand.
//
// TODO_TECHDEBT(@bryanchriswhite): Move the pre-mind relays in 'pkg/relayer/miner/relay_fixtures_test.go'
// to 'testutil', making any necessary adjustments the utils or docs as well.
func NewUnsignedMinedRelay(
	t *testing.T,
	session *sessiontypes.Session,
	supplierOperatorAddress string,
) *relayer.MinedRelay {
	t.Helper()

	relay := servicetypes.Relay{
		Req: &servicetypes.RelayRequest{
			Meta: servicetypes.RelayRequestMetadata{
				SessionHeader:           session.Header,
				SupplierOperatorAddress: supplierOperatorAddress,
			},
			Payload: []byte("request_payload"),
		},
		Res: &servicetypes.RelayResponse{
			Meta: servicetypes.RelayResponseMetadata{
				SessionHeader: session.Header,
			},
			Payload: []byte("response_payload"),
		},
	}

	relayBz, err := relay.Marshal()
	require.NoError(t, err)

	relayHashArr := protocol.GetRelayHashFromBytes(relayBz)
	relayHash := relayHashArr[:]

	return &relayer.MinedRelay{
		Relay: relay,
		Bytes: relayBz,
		Hash:  relayHash,
	}
}

// NewSignedMinedRelay returns a new "mined relay" with the given session data,
// as well as the bytes and the hash fields populated.
//
// IT DOES populate the signature fields and should only be used in contexts
// where a fully signed mined relay is needed for testing purposes.
//
// TODO_IMPROVE: It does not (yet) verify against and adhere to the actual
// relay mining difficulty of the service at hand.
//
// TODO_TECHDEBT(@bryanchriswhite): Move the pre-mind relays in 'pkg/relayer/miner/relay_fixtures_test.go'
// to 'testutil', making any necessary adjustments the utils or docs as well.
func NewSignedMinedRelay(
	t *testing.T,
	ctx context.Context,
	session *sessiontypes.Session,
	appAddr, supplierOperatorAddr, supplierOperatorKeyUid string,
	keyRing keyring.Keyring,
	ringClient crypto.RingClient,
) *relayer.MinedRelay {
	t.Helper()

	relay := servicetypes.Relay{
		Req: &servicetypes.RelayRequest{
			Meta: servicetypes.RelayRequestMetadata{
				SessionHeader:           session.Header,
				SupplierOperatorAddress: supplierOperatorAddr,
			},
			Payload: randomPayload(),
		},
		Res: &servicetypes.RelayResponse{
			Meta: servicetypes.RelayResponseMetadata{
				SessionHeader: session.Header,
			},
			Payload: randomPayload(),
		},
	}

	SignRelayRequest(ctx, t, &relay, appAddr, keyRing, ringClient)
	SignRelayResponse(ctx, t, &relay, supplierOperatorKeyUid, supplierOperatorAddr, keyRing)

	// TODO_TECHDEBT(@red-0ne): marshal using canonical codec.
	relayBz, err := relay.Marshal()
	require.NoError(t, err)

	relayHashArr := protocol.GetRelayHashFromBytes(relayBz)
	relayHash := relayHashArr[:]

	return &relayer.MinedRelay{
		Relay: relay,
		Bytes: relayBz,
		Hash:  relayHash,
	}
}

// TODO_TECHDEBT(@red-0ne): Centralize this logic in the relayer package.
// SignRelayRequest signs the relay request (updates relay.Req.Meta.Signature)
// on behalf of appAddr using the clients provided.
func SignRelayRequest(
	ctx context.Context,
	t *testing.T,
	relay *servicetypes.Relay,
	appAddr string,
	keyRing keyring.Keyring,
	ringClient crypto.RingClient,
) {
	t.Helper()

	relayReqMeta := relay.GetReq().GetMeta()
	sessionEndHeight := relayReqMeta.GetSessionHeader().GetSessionEndBlockHeight()

	// Retrieve the signing ring associated with the application address at the session end height.
	appRing, err := ringClient.GetRingForAddressAtHeight(ctx, appAddr, sessionEndHeight)
	require.NoError(t, err)

	// Retrieve the signing key associated with the application address.
	signingKey := testutilkeyring.GetSigningKeyFromAddress(t,
		appAddr,
		keyRing,
	)

	// Retrieve the signable bytes for the relay request.
	relayReqSignableBz, err := relay.GetReq().GetSignableBytesHash()
	require.NoError(t, err)

	// Sign the relay request.
	signature, err := appRing.Sign(relayReqSignableBz, signingKey)
	require.NoError(t, err)

	// Serialize the signature.
	signatureBz, err := signature.Serialize()
	require.NoError(t, err)

	// Update the relay request signature.
	relay.Req.Meta.Signature = signatureBz
}

// TODO_TECHDEBT(@red-0ne): Centralize this logic in the relayer package.
// in the relayer package?
// SignRelayResponse signs the relay response (updates relay.Res.Meta.SupplierOperatorSignature)
// on behalf of supplierOperatorAddr using the clients provided.
func SignRelayResponse(
	_ context.Context,
	t *testing.T,
	relay *servicetypes.Relay,
	supplierOperatorKeyUid, supplierOperatorAddr string,
	keyRing keyring.Keyring,
) {
	t.Helper()

	// Retrieve ths signable bytes for the relay response.
	relayResSignableBz, err := relay.GetRes().GetSignableBytesHash()
	require.NoError(t, err)

	// Sign the relay response.
	signatureBz, signerPubKey, err := keyRing.Sign(supplierOperatorKeyUid, relayResSignableBz[:], signingtypes.SignMode_SIGN_MODE_DIRECT)
	require.NoError(t, err)

	// Verify the signer address matches the expected supplier operator address.
	addr, err := cosmostypes.AccAddressFromBech32(supplierOperatorAddr)
	require.NoError(t, err)
	addrHexBz := strings.ToUpper(fmt.Sprintf("%x", addr.Bytes()))
	require.Equal(t, addrHexBz, signerPubKey.Address().String())

	// Update the relay response signature.
	relay.Res.Meta.SupplierOperatorSignature = signatureBz
}

// NewSignedEmptyRelay creates a new relay structure for the given req & res headers.
// It signs the relay request on behalf of application in the reqHeader.
// It signs the relay response on behalf of supplier provided..
func NewSignedEmptyRelay(
	ctx context.Context,
	t *testing.T,
	supplierOperatorKeyUid, supplierOperatorAddr string,
	reqHeader, resHeader *sessiontypes.SessionHeader,
	keyRing keyring.Keyring,
	ringClient crypto.RingClient,
) *servicetypes.Relay {
	t.Helper()

	relay := NewEmptyRelay(reqHeader, resHeader, supplierOperatorAddr)
	SignRelayRequest(ctx, t, relay, reqHeader.GetApplicationAddress(), keyRing, ringClient)
	SignRelayResponse(ctx, t, relay, supplierOperatorKeyUid, supplierOperatorAddr, keyRing)

	return relay
}

// NewEmptyRelay creates a new relay structure for the given req & res headers
// WITHOUT any payload or signatures.
func NewEmptyRelay(reqHeader, resHeader *sessiontypes.SessionHeader, supplierOperatorAddr string) *servicetypes.Relay {
	return &servicetypes.Relay{
		Req: &servicetypes.RelayRequest{
			Meta: servicetypes.RelayRequestMetadata{
				SessionHeader:           reqHeader,
				Signature:               nil, // Signature added elsewhere.
				SupplierOperatorAddress: supplierOperatorAddr,
			},
			Payload: nil,
		},
		Res: &servicetypes.RelayResponse{
			Meta: servicetypes.RelayResponseMetadata{
				SessionHeader:             resHeader,
				SupplierOperatorSignature: nil, // Signature added elsewhere.
			},
			Payload: nil,
		},
	}
}

const (
	charset       = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	payloadLength = 32
)

func randomPayload() []byte {
	rand.Seed(uint64(time.Now().UnixNano()))
	bz := make([]byte, payloadLength)
	for i := range bz {
		bz[i] = charset[rand.Intn(len(charset))]
	}
	return bz
}
