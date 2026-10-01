package rings

import (
	"context"
	"crypto/sha3"
	"encoding/binary"
	"testing"

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

// For an application with no delegated gateways, the expected ring contains
// only the app's own key.
func TestGetRingForAddress_NoGateway_RingIsAppKeyOnly(t *testing.T) {
	appAddr, appPubKey := sample.AccAddressAndPubKey()

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

	rc := &ringClient{
		logger:             polyzero.NewLogger(),
		applicationQuerier: appQuerier,
		accountQuerier:     acctQuerier,
		sharedQuerier:      sharedQuerier,
	}

	expectedPoints, err := rc.getRingPointsForAddressAtHeight(context.Background(), appAddr, 1)
	require.NoError(t, err)

	// The ring contains exactly the app's own key.
	appPoints, err := pointsFromPublicKeys(appPubKey)
	require.NoError(t, err)
	require.Contains(t, expectedPoints, string(appPoints[0].Encode()))
	require.Len(t, expectedPoints, 1)
}
