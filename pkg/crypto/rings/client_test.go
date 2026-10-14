package rings

import (
	"context"
	"crypto/sha3"
	"encoding/binary"
	"testing"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	ring "github.com/pokt-network/ring-go"
	"github.com/stretchr/testify/require"

	"go.uber.org/mock/gomock"

	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/testutil/mockclient"
	"github.com/pokt-network/poktroll/testutil/sample"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
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

	appAddr, appPubKey := sample.AccAddressAndPubKey()
	rc := newTestRingClient(t, appAddr, appPubKey)
	relayRequest := &servicetypes.RelayRequest{
		Meta: servicetypes.RelayRequestMetadata{
			SessionHeader: &sessiontypes.SessionHeader{
				ApplicationAddress:      appAddr,
				SessionId:               "session",
				ServiceId:               "svc",
				SessionStartBlockHeight: 1,
				SessionEndBlockHeight:   2,
			},
			Signature: undersizedSigBz,
		},
	}

	err = rc.VerifyRelayRequestSignature(context.Background(), relayRequest)
	require.ErrorContains(t, err, "expected between 2 and")
}

// For an application with no delegated gateways, the expected ring contains
// only the app's own key.
func TestGetRingForAddress_NoGateway_RingIsAppKeyOnly(t *testing.T) {
	appAddr, appPubKey := sample.AccAddressAndPubKey()

	rc := newTestRingClient(t, appAddr, appPubKey)

	expectedPoints, err := rc.getRingPointsForAddressAtHeight(context.Background(), appAddr, 1)
	require.NoError(t, err)

	// The ring contains exactly the app's own key.
	appPoints, err := pointsFromPublicKeys(appPubKey)
	require.NoError(t, err)
	require.Contains(t, expectedPoints, string(appPoints[0].Encode()))
	require.Len(t, expectedPoints, 1)
}

// A serialized ring signature must match the ring-go layout for its member count.
func TestValidateRingSignatureSize(t *testing.T) {
	curve := ring.Secp256k1()
	priv := curve.NewRandomScalar()
	keyRing, err := ring.NewKeyRing(curve, 3, priv, 0)
	require.NoError(t, err)
	msg := sha3.Sum256([]byte("m"))
	sig, err := keyRing.Sign(msg, priv)
	require.NoError(t, err)
	validSigBz, err := sig.Serialize()
	require.NoError(t, err)
	require.NoError(t, validateRingSignatureSize(validSigBz, 3))

	oneMemberSigBz := make([]byte, 4+32+33+(32+33))
	binary.BigEndian.PutUint32(oneMemberSigBz[:4], 1)

	tests := map[string][]byte{
		"empty":           {},
		"one byte":        {0x00},
		"three bytes":     {0x00, 0x00, 0x02},
		"size only":       validSigBz[:4],
		"one member":      oneMemberSigBz,
		"truncated":       validSigBz[:len(validSigBz)-1],
		"trailing byte":   append(append([]byte{}, validSigBz...), 0x00),
		"huge member set": append([]byte{0xff, 0xff, 0xff, 0xff}, validSigBz[4:]...),
	}
	for name, sigBz := range tests {
		t.Run(name, func(t *testing.T) {
			require.ErrorIs(t, validateRingSignatureSize(sigBz, 3), ErrRingClientInvalidRelayRequestSignature)
		})
	}

	// The member count must not exceed the allowed maximum.
	require.ErrorIs(t, validateRingSignatureSize(validSigBz, 2), ErrRingClientInvalidRelayRequestSignature)
}

// The maximum ring size admits every ring an honest signer can build.
func TestMaxRingSignatureMembers(t *testing.T) {
	tests := []struct {
		desc                 string
		maxDelegatedGateways uint64
		honestRingSize       int
	}{
		{desc: "no delegations: the app twice", maxDelegatedGateways: 1, honestRingSize: 2},
		{desc: "max delegations", maxDelegatedGateways: 7, honestRingSize: 1 + 7},
		{desc: "max delegations plus as many undelegated since the session", maxDelegatedGateways: 7, honestRingSize: 1 + 2*7},
		{desc: "more delegations than a lowered max", maxDelegatedGateways: 2, honestRingSize: 1 + 7},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			// The verifier's expected ring is at least as large as the signer's.
			maxMembers := maxRingSignatureMembers(test.maxDelegatedGateways, test.honestRingSize)
			require.GreaterOrEqual(t, maxMembers, uint64(test.honestRingSize))
		})
	}
	require.Equal(t, uint64(15), maxRingSignatureMembers(7, 2))
}

// newTestRingClient returns a ring client for an application with no delegated
// gateways.
func newTestRingClient(t *testing.T, appAddr string, appPubKey cryptotypes.PubKey) *ringClient {
	t.Helper()
	ctrl := gomock.NewController(t)
	appQuerier := mockclient.NewMockApplicationQueryClient(ctrl)
	appQuerier.EXPECT().GetApplication(gomock.Any(), appAddr).Return(
		apptypes.Application{Address: appAddr}, // no DelegateeGatewayAddresses
		nil,
	).AnyTimes()
	appParams := apptypes.DefaultParams()
	appQuerier.EXPECT().GetParams(gomock.Any()).Return(&appParams, nil).AnyTimes()

	acctQuerier := mockclient.NewMockAccountQueryClient(ctrl)
	acctQuerier.EXPECT().GetPubKeyFromAddress(gomock.Any(), appAddr).Return(appPubKey, nil).AnyTimes()

	sharedParams := sharedtypes.DefaultParams()
	sharedQuerier := mockclient.NewMockSharedQueryClient(ctrl)
	sharedQuerier.EXPECT().GetParams(gomock.Any()).Return(&sharedParams, nil).AnyTimes()

	return &ringClient{
		logger:             polyzero.NewLogger(),
		applicationQuerier: appQuerier,
		accountQuerier:     acctQuerier,
		sharedQuerier:      sharedQuerier,
	}
}

// Malformed relay requests are rejected with an error, without panicking.
func TestVerifyRelayRequestSignature_MalformedRequests(t *testing.T) {
	appAddr, appPubKey := sample.AccAddressAndPubKey()
	rc := newTestRingClient(t, appAddr, appPubKey)
	for desc, relayRequest := range map[string]*servicetypes.RelayRequest{
		"nil request":       nil,
		"empty request":     {},
		"empty header":      {Meta: servicetypes.RelayRequestMetadata{SessionHeader: &sessiontypes.SessionHeader{}}},
		"no signature":      {Meta: servicetypes.RelayRequestMetadata{SessionHeader: &sessiontypes.SessionHeader{ApplicationAddress: appAddr, SessionId: "s", ServiceId: "svc", SessionStartBlockHeight: 1, SessionEndBlockHeight: 2}}},
		"garbage signature": {Meta: servicetypes.RelayRequestMetadata{SessionHeader: &sessiontypes.SessionHeader{ApplicationAddress: appAddr, SessionId: "s", ServiceId: "svc", SessionStartBlockHeight: 1, SessionEndBlockHeight: 2}, Signature: []byte{0, 0, 0, 2, 1}}},
	} {
		t.Run(desc, func(t *testing.T) {
			require.Error(t, rc.VerifyRelayRequestSignature(context.Background(), relayRequest))
		})
	}
}
