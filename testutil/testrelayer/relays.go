package testrelayer

import (
	"context"
	"fmt"
	"strings"
	"testing"

	ring_secp256k1 "github.com/athanorlabs/go-dleq/secp256k1"
	ringtypes "github.com/athanorlabs/go-dleq/types"
	cosmoscrypto "github.com/cosmos/cosmos-sdk/crypto"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/crypto"
	"github.com/pokt-network/poktroll/pkg/relayer"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

// NewUnsignedMinedRelay returns a new mined relay with the given session data,
// as well as the bytes and the hash fields populated.
//
// It DOES NOT populate the signature fields and should only be used in contexts
// where a partial mined relay is enough for testing purposes.
func NewUnsignedMinedRelay(
	t *testing.T,
	session *sessiontypes.Session,
	supplierAddress string,
) *relayer.MinedRelay {
	t.Helper()

	relay := servicetypes.Relay{
		Req: &servicetypes.RelayRequest{
			Meta: servicetypes.RelayRequestMetadata{
				SessionHeader:   session.Header,
				SupplierAddress: supplierAddress,
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

// NewSignedMinedRelay returns a new mined relay with the given session data,
// as well as the bytes and the hash fields populated.
//
// IT DOES populate the signature fields and should only be used in contexts
// where a fully signed mined relay is needed for testing purposes.
func NewSignedMinedRelay(
	t *testing.T,
	ctx context.Context,
	session *sessiontypes.Session,
	appAddr, supplierAddr, supplierKeyUid string,
	keyRing keyring.Keyring,
	ringClient crypto.RingClient,
) *relayer.MinedRelay {
	t.Helper()

	relay := servicetypes.Relay{
		Req: &servicetypes.RelayRequest{
			Meta: servicetypes.RelayRequestMetadata{
				SessionHeader:   session.Header,
				SupplierAddress: supplierAddr,
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

	SignRelayRequest(ctx, t, &relay, appAddr, keyRing, ringClient)
	SignRelayResponse(ctx, t, &relay, supplierKeyUid, supplierAddr, keyRing)

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
	signingKey := GetSigningKeyFromAddress(t,
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
// SignRelayResponse signs the relay response (updates relay.Res.Meta.SupplierSignature)
// on behalf of supplierAddr using the clients provided.
func SignRelayResponse(
	_ context.Context,
	t *testing.T,
	relay *servicetypes.Relay,
	supplierKeyUid, supplierAddr string,
	keyRing keyring.Keyring,
) {
	t.Helper()

	// Retrieve ths signable bytes for the relay response.
	relayResSignableBz, err := relay.GetRes().GetSignableBytesHash()
	require.NoError(t, err)

	// Sign the relay response.
	signatureBz, signerPubKey, err := keyRing.Sign(supplierKeyUid, relayResSignableBz[:], signingtypes.SignMode_SIGN_MODE_DIRECT)
	require.NoError(t, err)

	// Verify the signer address matches the expected supplier address.
	addr, err := cosmostypes.AccAddressFromBech32(supplierAddr)
	require.NoError(t, err)
	addrHexBz := strings.ToUpper(fmt.Sprintf("%x", addr.Bytes()))
	require.Equal(t, addrHexBz, signerPubKey.Address().String())

	// Update the relay response signature.
	relay.Res.Meta.SupplierSignature = signatureBz
}

// GetSigningKeyFromAddress retrieves the signing key associated with the given
// bech32 address from the provided keyring.
func GetSigningKeyFromAddress(t *testing.T, bech32 string, keyRing keyring.Keyring) ringtypes.Scalar {
	t.Helper()

	addr, err := cosmostypes.AccAddressFromBech32(bech32)
	require.NoError(t, err)

	armorPrivKey, err := keyRing.ExportPrivKeyArmorByAddress(addr, "")
	require.NoError(t, err)

	privKey, _, err := cosmoscrypto.UnarmorDecryptPrivKey(armorPrivKey, "")
	require.NoError(t, err)

	curve := ring_secp256k1.NewCurve()
	signingKey, err := curve.DecodeToScalar(privKey.Bytes())
	require.NoError(t, err)

	return signingKey
}

// NewSignedEmptyRelay creates a new relay structure for the given req & res headers.
// It signs the relay request on behalf of application in the reqHeader.
// It signs the relay response on behalf of supplier provided..
func NewSignedEmptyRelay(
	ctx context.Context,
	t *testing.T,
	supplierKeyUid, supplierAddr string,
	reqHeader, resHeader *sessiontypes.SessionHeader,
	keyRing keyring.Keyring,
	ringClient crypto.RingClient,
) *servicetypes.Relay {
	t.Helper()

	relay := NewEmptyRelay(reqHeader, resHeader, supplierAddr)
	SignRelayRequest(ctx, t, relay, reqHeader.GetApplicationAddress(), keyRing, ringClient)
	SignRelayResponse(ctx, t, relay, supplierKeyUid, supplierAddr, keyRing)

	return relay
}

// NewEmptyRelay creates a new relay structure for the given req & res headers
// WITHOUT any payload or signatures.
func NewEmptyRelay(reqHeader, resHeader *sessiontypes.SessionHeader, supplierAddr string) *servicetypes.Relay {
	return &servicetypes.Relay{
		Req: &servicetypes.RelayRequest{
			Meta: servicetypes.RelayRequestMetadata{
				SessionHeader:   reqHeader,
				Signature:       nil, // Signature added elsewhere.
				SupplierAddress: supplierAddr,
			},
			Payload: nil,
		},
		Res: &servicetypes.RelayResponse{
			Meta: servicetypes.RelayResponseMetadata{
				SessionHeader:     resHeader,
				SupplierSignature: nil, // Signature added elsewhere.
			},
			Payload: nil,
		},
	}
}
