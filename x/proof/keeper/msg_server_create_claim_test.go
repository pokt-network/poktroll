package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/proof"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/proof/keeper"
	"github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestMsgServer_CreateClaim_Success(t *testing.T) {
	appSupplierPair := proof.AppSupplierPair{
		AppAddr:      sample.AccAddress(),
		SupplierAddr: sample.AccAddress(),
	}
	service := &sharedtypes.Service{Id: testServiceId}
	sessionFixturesByAddr := proof.NewSessionFixturesWithPairings(t, service, appSupplierPair)

	proofKeeper, ctx := keepertest.ProofKeeper(t, sessionFixturesByAddr)
	srv := keeper.NewMsgServerImpl(proofKeeper)

	claimMsg := newTestClaimMsg(t, testSessionId)
	claimMsg.SupplierAddress = appSupplierPair.SupplierAddr
	claimMsg.SessionHeader.ApplicationAddress = appSupplierPair.AppAddr

	createClaimRes, err := srv.CreateClaim(ctx, claimMsg)
	require.NoError(t, err)
	require.NotNil(t, createClaimRes)

	claimRes, err := proofKeeper.AllClaims(ctx, &types.QueryAllClaimsRequest{})
	require.NoError(t, err)

	claims := claimRes.GetClaims()
	require.Lenf(t, claims, 1, "expected 1 claim, got %d", len(claims))
	require.Equal(t, claimMsg.SessionHeader.SessionId, claims[0].GetSessionHeader().GetSessionId())
	require.Equal(t, claimMsg.SupplierAddress, claims[0].GetSupplierAddress())
	require.Equal(t, claimMsg.SessionHeader.GetSessionEndBlockHeight(), claims[0].GetSessionHeader().GetSessionEndBlockHeight())
	require.Equal(t, claimMsg.RootHash, claims[0].GetRootHash())
}

func TestMsgServer_CreateClaim_Error(t *testing.T) {
	service := &sharedtypes.Service{Id: testServiceId}
	appSupplierPair := proof.AppSupplierPair{
		AppAddr:      sample.AccAddress(),
		SupplierAddr: sample.AccAddress(),
	}
	sessionFixturesByAppAddr := proof.NewSessionFixturesWithPairings(t, service, appSupplierPair)

	proofKeeper, ctx := keepertest.ProofKeeper(t, sessionFixturesByAppAddr)
	srv := keeper.NewMsgServerImpl(proofKeeper)

	tests := []struct {
		desc        string
		claimMsgFn  func(t *testing.T) *types.MsgCreateClaim
		expectedErr error
	}{
		{
			desc: "on-chain session ID must match claim msg session ID",
			claimMsgFn: func(t *testing.T) *types.MsgCreateClaim {
				msg := newTestClaimMsg(t, "invalid_session_id")
				msg.SupplierAddress = appSupplierPair.SupplierAddr
				msg.SessionHeader.ApplicationAddress = appSupplierPair.AppAddr

				return msg
			},
			expectedErr: status.Error(
				codes.InvalidArgument,
				types.ErrProofInvalidSessionId.Wrapf(
					"session ID does not match on-chain session ID; expected %q, got %q",
					testSessionId,
					"invalid_session_id",
				).Error(),
			),
		},
		{
			desc: "claim msg supplier address must be in the session",
			claimMsgFn: func(t *testing.T) *types.MsgCreateClaim {
				msg := newTestClaimMsg(t, testSessionId)
				msg.SessionHeader.ApplicationAddress = appSupplierPair.AppAddr

				// Overwrite supplier address to one not included in the session fixtures.
				msg.SupplierAddress = sample.AccAddress()

				return msg
			},
			expectedErr: types.ErrProofNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			createClaimRes, err := srv.CreateClaim(ctx, test.claimMsgFn(t))
			require.ErrorContains(t, err, test.expectedErr.Error())
			require.Nil(t, createClaimRes)
		})
	}
}

func newTestClaimMsg(t *testing.T, sessionId string) *types.MsgCreateClaim {
	t.Helper()

	return types.NewMsgCreateClaim(
		sample.AccAddress(),
		&sessiontypes.SessionHeader{
			ApplicationAddress:      sample.AccAddress(),
			SessionStartBlockHeight: 0,
			SessionId:               sessionId,
			Service: &sharedtypes.Service{
				Id:   "svc1",
				Name: "svc1",
			},
		},
		[]byte{0, 0, 0, 0},
	)
}
