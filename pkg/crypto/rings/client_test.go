package rings

import (
	"context"
	"crypto/sha3"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	ringtypes "github.com/pokt-network/go-dleq/types"
	ring "github.com/pokt-network/ring-go"
	"github.com/stretchr/testify/require"

	"go.uber.org/mock/gomock"

	"github.com/pokt-network/poktroll/pkg/client/query"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/testutil/mockclient"
	"github.com/pokt-network/poktroll/testutil/sample"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
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

	expectedPoints, _, err := rc.getRingPointsForAddressAtSessionEndHeight(context.Background(), appAddr, 1)
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

// Rebuilding a past ring must not write into the application's delegations.
func TestGetRingAddressesAtSessionEndHeight_DoesNotModifyApplication(t *testing.T) {
	delegations := make([]string, 1, 2)
	delegations[0] = "gw_delegated"
	app := &apptypes.Application{
		DelegateeGatewayAddresses: delegations,
		PendingUndelegations: map[uint64]apptypes.UndelegatingGatewayList{
			20: {GatewayAddresses: []string{"gw_undelegated"}},
		},
	}

	ringAddresses := GetRingAddressesAtSessionEndHeight(app, 10)
	require.ElementsMatch(t, []string{"gw_delegated", "gw_undelegated"}, ringAddresses)
	require.Empty(t, delegations[:cap(delegations)][1])
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

// Signature bytes that pass the size check but fail later are rejected with an
// error, without panicking.
func TestVerifyRingSignature_InvalidBytesReturnError(t *testing.T) {
	curve := ring.Secp256k1()
	priv := curve.NewRandomScalar()
	keyRing, err := ring.NewKeyRing(curve, 2, priv, 0)
	require.NoError(t, err)
	msg := sha3.Sum256([]byte("m"))
	sig, err := keyRing.Sign(msg, priv)
	require.NoError(t, err)
	validSigBz, err := sig.Serialize()
	require.NoError(t, err)

	garbageSigBz := make([]byte, len(validSigBz))
	copy(garbageSigBz, validSigBz[:4])
	for i := 4; i < len(garbageSigBz); i++ {
		garbageSigBz[i] = 0xff
	}
	require.NoError(t, validateRingSignatureSize(garbageSigBz, MaxRingSignatureMembers))

	ringPoints := make(map[string]ringtypes.Point)
	for _, publicKey := range sig.PublicKeys() {
		ringPoints[string(publicKey.Encode())] = publicKey
	}

	err = verifyRingSignature(garbageSigBz, ringPoints, &servicetypes.RelayRequest{}, "app")
	require.ErrorIs(t, err, ErrRingClientInvalidRelayRequestSignature)

	// A nil request panics while hashing the signable bytes.
	err = verifyRingSignature(validSigBz, ringPoints, nil, "app")
	require.ErrorContains(t, err, "panic verifying ring signature")
}

// The ring for a relay request is the one of the request's session, at its
// session end height, whatever the current session length is.
func TestGetRingPointsForAddressAtSessionEndHeight_UsesSessionEndHeight(t *testing.T) {
	appAddr, appPubKey := sample.AccAddressAndPubKey()
	gatewayAddr, gatewayPubKey := sample.AccAddressAndPubKey()
	const sessionEndHeight = 10

	// The gateway was undelegated in the session ending at sessionEndHeight,
	// so it is still part of that session's ring.
	ctrl := gomock.NewController(t)
	appQuerier := mockclient.NewMockApplicationQueryClient(ctrl)
	appQuerier.EXPECT().GetApplication(gomock.Any(), appAddr).Return(
		apptypes.Application{
			Address: appAddr,
			PendingUndelegations: map[uint64]apptypes.UndelegatingGatewayList{
				sessionEndHeight: {GatewayAddresses: []string{gatewayAddr}},
			},
		},
		nil,
	).AnyTimes()
	acctQuerier := mockclient.NewMockAccountQueryClient(ctrl)
	acctQuerier.EXPECT().GetPubKeyFromAddress(gomock.Any(), appAddr).Return(appPubKey, nil).AnyTimes()
	acctQuerier.EXPECT().GetPubKeyFromAddress(gomock.Any(), gatewayAddr).Return(gatewayPubKey, nil).AnyTimes()

	// The current session length puts sessionEndHeight inside a later session.
	liveParams := sharedtypes.DefaultParams()
	liveParams.NumBlocksPerSession = 4
	require.Greater(t, sharedtypes.GetSessionEndHeight(&liveParams, sessionEndHeight), int64(sessionEndHeight))
	sharedQuerier := mockclient.NewMockSharedQueryClient(ctrl)
	sharedQuerier.EXPECT().GetParams(gomock.Any()).Return(&liveParams, nil).AnyTimes()

	rc := &ringClient{
		logger:             polyzero.NewLogger(),
		applicationQuerier: appQuerier,
		accountQuerier:     acctQuerier,
		sharedQuerier:      sharedQuerier,
	}

	ringPoints, _, err := rc.getRingPointsForAddressAtSessionEndHeight(context.Background(), appAddr, sessionEndHeight)
	require.NoError(t, err)
	gatewayPoints, err := pointsFromPublicKeys(gatewayPubKey)
	require.NoError(t, err)
	require.Contains(t, ringPoints, string(gatewayPoints[0].Encode()))
}

// A relay request signature cannot have more members than the expected ring.
func TestVerifyRelayRequestSignature_RejectsMoreMembersThanExpectedRing(t *testing.T) {
	appPrivKey := secp256k1.GenPrivKey()
	appPubKey := appPrivKey.PubKey()
	appAddr := cosmostypes.AccAddress(appPubKey.Address()).String()
	appScalar, err := ring.Secp256k1().DecodeToScalar(appPrivKey.Bytes())
	require.NoError(t, err)

	rc := newTestRingClient(t, appAddr, appPubKey)

	signRelayRequest := func(numMembers int) *servicetypes.RelayRequest {
		pubKeys := make([]cryptotypes.PubKey, numMembers)
		for i := range pubKeys {
			pubKeys[i] = appPubKey
		}
		signingRing, ringErr := GetRingFromPubKeys(pubKeys)
		require.NoError(t, ringErr)

		relayRequest := &servicetypes.RelayRequest{
			Meta: servicetypes.RelayRequestMetadata{
				SessionHeader: &sessiontypes.SessionHeader{
					ApplicationAddress:      appAddr,
					SessionId:               "session",
					ServiceId:               "svc",
					SessionStartBlockHeight: 1,
					SessionEndBlockHeight:   2,
				},
			},
		}
		signableBz, hashErr := relayRequest.GetSignableBytesHash()
		require.NoError(t, hashErr)
		sig, signErr := signingRing.Sign(signableBz, appScalar)
		require.NoError(t, signErr)
		sigBz, serializeErr := sig.Serialize()
		require.NoError(t, serializeErr)
		relayRequest.Meta.Signature = sigBz
		return relayRequest
	}

	// The ring of an application without gateways has two members.
	require.NoError(t, rc.VerifyRelayRequestSignature(context.Background(), signRelayRequest(2)))

	err = rc.VerifyRelayRequestSignature(context.Background(), signRelayRequest(4))
	require.ErrorIs(t, err, ErrRingClientInvalidRelayRequestSignature)
}

// A ring that lists a key more than once, as built by the ring client itself,
// is accepted: here the application is also one of its own delegated gateways.
func TestVerifyRelayRequestSignature_AcceptsRingWithRepeatedKey(t *testing.T) {
	appPrivKey := secp256k1.GenPrivKey()
	appPubKey := appPrivKey.PubKey()
	appAddr := cosmostypes.AccAddress(appPubKey.Address()).String()
	appScalar, err := ring.Secp256k1().DecodeToScalar(appPrivKey.Bytes())
	require.NoError(t, err)
	gatewayAddr, gatewayPubKey := sample.AccAddressAndPubKey()

	ctrl := gomock.NewController(t)
	appQuerier := mockclient.NewMockApplicationQueryClient(ctrl)
	appQuerier.EXPECT().GetApplication(gomock.Any(), appAddr).Return(
		apptypes.Application{Address: appAddr, DelegateeGatewayAddresses: []string{appAddr, gatewayAddr}},
		nil,
	).AnyTimes()
	acctQuerier := mockclient.NewMockAccountQueryClient(ctrl)
	acctQuerier.EXPECT().GetPubKeyFromAddress(gomock.Any(), appAddr).Return(appPubKey, nil).AnyTimes()
	acctQuerier.EXPECT().GetPubKeyFromAddress(gomock.Any(), gatewayAddr).Return(gatewayPubKey, nil).AnyTimes()
	sharedParams := sharedtypes.DefaultParams()
	sharedQuerier := mockclient.NewMockSharedQueryClient(ctrl)
	sharedQuerier.EXPECT().GetParams(gomock.Any()).Return(&sharedParams, nil).AnyTimes()
	rc := &ringClient{
		logger:             polyzero.NewLogger(),
		applicationQuerier: appQuerier,
		accountQuerier:     acctQuerier,
		sharedQuerier:      sharedQuerier,
	}

	sessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, 1)
	signingRing, err := rc.GetRingForAddressAtHeight(context.Background(), appAddr, sessionEndHeight)
	require.NoError(t, err)
	require.Equal(t, 3, signingRing.Size())

	relayRequest := &servicetypes.RelayRequest{
		Meta: servicetypes.RelayRequestMetadata{
			SessionHeader: &sessiontypes.SessionHeader{
				ApplicationAddress:      appAddr,
				SessionId:               "session",
				ServiceId:               "svc",
				SessionStartBlockHeight: 1,
				SessionEndBlockHeight:   sessionEndHeight,
			},
		},
	}
	signableBz, err := relayRequest.GetSignableBytesHash()
	require.NoError(t, err)
	sig, err := signingRing.Sign(signableBz, appScalar)
	require.NoError(t, err)
	relayRequest.Meta.Signature, err = sig.Serialize()
	require.NoError(t, err)

	require.NoError(t, rc.VerifyRelayRequestSignature(context.Background(), relayRequest))
}

// The application's own key is required: a ring is not built without it.
func TestGetRingPubKeysForAddress_AppKeyRequired(t *testing.T) {
	appAddr := sample.AccAddressBech32()
	ctrl := gomock.NewController(t)
	appQuerier := mockclient.NewMockApplicationQueryClient(ctrl)
	appQuerier.EXPECT().GetApplication(gomock.Any(), appAddr).Return(apptypes.Application{Address: appAddr}, nil).AnyTimes()
	acctQuerier := mockclient.NewMockAccountQueryClient(ctrl)
	acctQuerier.EXPECT().GetPubKeyFromAddress(gomock.Any(), appAddr).Return(nil, errors.New("no key")).AnyTimes()

	rc := &ringClient{logger: polyzero.NewLogger(), applicationQuerier: appQuerier, accountQuerier: acctQuerier}
	pubKeys, err := rc.getRingPubKeysForAddress(context.Background(), appAddr, 1)
	require.Error(t, err)
	require.Nil(t, pubKeys)
}

// A gateway with no account or no public key is left out of the ring, as is a
// gateway whose key lookup fails with any error the onchain account client can
// return. A failed offchain account query is returned instead.
func TestGetRingPubKeysForAddress_GatewayKeyErrors(t *testing.T) {
	appAddr, appPubKey := sample.AccAddressAndPubKey()
	gatewayAddr := sample.AccAddressBech32()

	tests := []struct {
		desc       string
		gatewayErr error
		expectErr  error
	}{
		{desc: "offchain account not found", gatewayErr: query.ErrQueryAccountNotFound},
		{desc: "offchain public key not found", gatewayErr: query.ErrQueryPubKeyNotFound},
		{desc: "onchain account not found", gatewayErr: prooftypes.ErrProofAccNotFound},
		{desc: "onchain public key not found", gatewayErr: prooftypes.ErrProofPubKeyNotFound},
		{desc: "onchain invalid address", gatewayErr: errors.New("decoding bech32 failed")},
		{desc: "offchain account query failed", gatewayErr: query.ErrQueryAccountQueryFailed, expectErr: query.ErrQueryAccountQueryFailed},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			appQuerier := mockclient.NewMockApplicationQueryClient(ctrl)
			appQuerier.EXPECT().GetApplication(gomock.Any(), appAddr).Return(
				apptypes.Application{Address: appAddr, DelegateeGatewayAddresses: []string{gatewayAddr}}, nil,
			).AnyTimes()
			acctQuerier := mockclient.NewMockAccountQueryClient(ctrl)
			acctQuerier.EXPECT().GetPubKeyFromAddress(gomock.Any(), appAddr).Return(appPubKey, nil).AnyTimes()
			acctQuerier.EXPECT().GetPubKeyFromAddress(gomock.Any(), gatewayAddr).
				Return(nil, test.gatewayErr).AnyTimes()

			rc := &ringClient{logger: polyzero.NewLogger(), applicationQuerier: appQuerier, accountQuerier: acctQuerier}
			pubKeys, err := rc.getRingPubKeysForAddress(context.Background(), appAddr, 1)
			if test.expectErr != nil {
				require.ErrorIs(t, err, test.expectErr)
				require.Nil(t, pubKeys)
				return
			}
			require.NoError(t, err)
			require.Equal(t, []cryptotypes.PubKey{appPubKey, appPubKey}, pubKeys)
		})
	}
}
