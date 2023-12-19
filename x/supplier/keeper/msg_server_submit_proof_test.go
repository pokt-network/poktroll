package keeper_test

import (
	"math/rand"
	"os"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/session"
	"github.com/pokt-network/poktroll/testutil/copy"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/supplier"
	"github.com/pokt-network/poktroll/testutil/testrelayer"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/keeper"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

var (
	testProofPath      = []byte{1, 0, 1, 0, 1, 0}
	wrongTestProofPath = []byte{0, 0, 0, 0, 1, 1}
)

func TestMsgServer_SubmitProof_Success(t *testing.T) {
	appSupplierPair := &supplier.AppSupplierPair{
		AppAddr:      sample.AccAddress(),
		SupplierAddr: sample.AccAddress(),
	}
	service := &sharedtypes.Service{Id: testServiceId}
	sessionFixturesByAppAddr := supplier.NewSessionFixturesWithPairings(t, service, appSupplierPair)
	sessionFixture := sessionFixturesByAppAddr[appSupplierPair.AppAddr]

	supplierKeeper, sdkCtx := keepertest.SupplierKeeper(t, sessionFixturesByAppAddr)
	claim, merkleTreeProofBz := ensureTestClaim(t, &sdkCtx, supplierKeeper, appSupplierPair, sessionFixture)

	srv := keeper.NewMsgServerImpl(*supplierKeeper)
	ctx := sdk.WrapSDKContext(sdkCtx)

	// Construct a valid submit proof message using the same supplier address and
	// session header as the corresponding claim.
	proofMsg := &suppliertypes.MsgSubmitProof{
		SupplierAddress: claim.GetSupplierAddress(),
		SessionHeader:   claim.GetSessionHeader(),
		Proof:           merkleTreeProofBz,
	}

	submitProofRes, err := srv.SubmitProof(ctx, proofMsg)
	require.NoError(t, err)
	require.NotNil(t, submitProofRes)

	proof, found := supplierKeeper.GetProof(
		sdkCtx,
		proofMsg.GetSessionHeader().GetSessionId(),
		proofMsg.GetSupplierAddress(),
	)
	require.Truef(
		t, found,
		"expected proof to be found for session ID %s and supplier address %s",
		proofMsg.GetSessionHeader().GetSessionId(),
		proofMsg.GetSupplierAddress(),
	)
	require.Equal(t, proofMsg.GetSupplierAddress(), proof.GetSupplierAddress())
	require.EqualValues(t, proofMsg.GetSessionHeader(), proof.GetSessionHeader())
}

func TestMsgServer_SubmitProof_Error(t *testing.T) {
	service := &sharedtypes.Service{Id: testServiceId}
	claimedAppSupplierPair := &supplier.AppSupplierPair{
		AppAddr:      sample.AccAddress(),
		SupplierAddr: sample.AccAddress(),
	}
	unClaimedAppSupplierPair := &supplier.AppSupplierPair{
		AppAddr:      sample.AccAddress(),
		SupplierAddr: claimedAppSupplierPair.SupplierAddr,
	}
	sessionFixturesByAppAddr := supplier.NewSessionFixturesWithPairings(
		t, service,
		claimedAppSupplierPair,
		unClaimedAppSupplierPair,
	)
	claimedSessionFixture := sessionFixturesByAppAddr[claimedAppSupplierPair.AppAddr]

	supplierKeeper, sdkCtx := keepertest.SupplierKeeper(t, sessionFixturesByAppAddr)
	claim, validClosestMerkleProofBz := ensureTestClaim(
		t, &sdkCtx,
		supplierKeeper,
		claimedAppSupplierPair,
		claimedSessionFixture,
	)
	require.Equal(t, claimedAppSupplierPair.SupplierAddr, claim.GetSupplierAddress())

	srv := keeper.NewMsgServerImpl(*supplierKeeper)
	ctx := sdk.WrapSDKContext(sdkCtx)

	claimedSessionHeader := claimedSessionFixture.GetHeader()

	tests := []struct {
		desc        string
		proofMsgFn  func(t *testing.T) *suppliertypes.MsgSubmitProof
		expectedErr error
	}{
		{
			desc: "proof msg application bech32 address is invalid",
			proofMsgFn: func(t *testing.T) *suppliertypes.MsgSubmitProof {

				invalidSessionHeaderApplicationBech32 := copy.DeepCopyJSON[sessiontypes.SessionHeader](t, *claimedSessionHeader)
				invalidSessionHeaderApplicationBech32.ApplicationAddress = "not_a_bech32_address"

				return &suppliertypes.MsgSubmitProof{
					SupplierAddress: claim.GetSupplierAddress(),
					SessionHeader:   &invalidSessionHeaderApplicationBech32,
					Proof:           validClosestMerkleProofBz,
				}
			},
			expectedErr: status.Error(
				codes.InvalidArgument,
				sdkerrors.ErrInvalidAddress.Wrapf(
					"application address: %q, error: %s",
					"not_a_bech32_address",
					"decoding bech32 failed: invalid separator index -1",
				).Error(),
			),
		},
		{
			desc: "proof msg session ID doesn't match on-chain session ID",
			proofMsgFn: func(t *testing.T) *suppliertypes.MsgSubmitProof {
				invalidSessionHeaderWrongSessionID := copy.DeepCopyJSON[sessiontypes.SessionHeader](t, *claimedSessionHeader)
				invalidSessionHeaderWrongSessionID.SessionId = "wrong_session_id"

				return &suppliertypes.MsgSubmitProof{
					SupplierAddress: claim.GetSupplierAddress(),
					SessionHeader:   &invalidSessionHeaderWrongSessionID,
					Proof:           validClosestMerkleProofBz,
				}
			},
			expectedErr: suppliertypes.ErrSupplierInvalidSessionId,
		},
		{
			desc: "proof session service ID doesn't match on-chain session",
			proofMsgFn: func(t *testing.T) *suppliertypes.MsgSubmitProof {
				// TODO_NEXT(@bryanchriswhite #141): move this test scenario to
				// a CLI test. This scenario is not possible with the current
				// keeper mock as it will always return a session matching the
				// session header in the request when calling #GetSession().
				t.SkipNow()
				return nil
			},
			expectedErr: status.Error(
				codes.InvalidArgument,
				suppliertypes.ErrSupplierInvalidSessionId.Wrapf(
					"",
				).Error(),
			),
		},
		{
			desc: "proof msg supplier address not in the on-chain session",
			proofMsgFn: func(t *testing.T) *suppliertypes.MsgSubmitProof {
				return &suppliertypes.MsgSubmitProof{
					SupplierAddress: sample.AccAddress(),
					SessionHeader:   claimedSessionHeader,
					Proof:           validClosestMerkleProofBz,
				}
			},
			expectedErr: suppliertypes.ErrSupplierNotFound,
		},
		{
			desc: "claim and proof session application address doesn't match",
			proofMsgFn: func(t *testing.T) *suppliertypes.MsgSubmitProof {
				unClaimedSessionFixture := sessionFixturesByAppAddr[unClaimedAppSupplierPair.AppAddr]
				return &suppliertypes.MsgSubmitProof{
					SupplierAddress: claim.GetSupplierAddress(),
					SessionHeader:   unClaimedSessionFixture.GetHeader(),
					Proof:           validClosestMerkleProofBz,
				}
			},
			expectedErr: suppliertypes.ErrSupplierInvalidApplicationAddress,
		},
		{
			desc: "claim and proof session application address don't match",
			proofMsgFn: func(t *testing.T) *suppliertypes.MsgSubmitProof {
				unClaimedSessionFixture := sessionFixturesByAppAddr[unClaimedAppSupplierPair.AppAddr]
				unclaimedSessionHeader := unClaimedSessionFixture.GetHeader()

				return &suppliertypes.MsgSubmitProof{
					SupplierAddress: claim.GetSupplierAddress(),
					SessionHeader:   unclaimedSessionHeader,
					Proof:           validClosestMerkleProofBz,
				}
			},
			expectedErr: status.Error(
				codes.FailedPrecondition,
				suppliertypes.ErrSupplierInvalidApplicationAddress.Wrapf(
					"claim application address %q does not match proof application address %q",
					claim.GetSessionHeader().GetApplicationAddress(),
					unClaimedAppSupplierPair.AppAddr,
				).Error(),
			),
		},
		{
			desc: "proof session service ID is empty",
			proofMsgFn: func(t *testing.T) *suppliertypes.MsgSubmitProof {
				invalidSessionHeaderEmptyServiceID := copy.DeepCopyJSON[sessiontypes.SessionHeader](t, *claimedSessionHeader)
				invalidSessionHeaderEmptyServiceID.Service.Id = ""

				return &suppliertypes.MsgSubmitProof{
					SupplierAddress: claim.GetSupplierAddress(),
					SessionHeader:   &invalidSessionHeaderEmptyServiceID,
					Proof:           validClosestMerkleProofBz,
				}
			},
			expectedErr: status.Error(
				codes.FailedPrecondition,
				suppliertypes.ErrSupplierInvalidServiceID.Wrapf(
					"claim service ID %q does not match proof service ID %q",
					claim.GetSessionHeader().GetService().GetId(),
					"",
				).Error(),
			),
		},
		{
			desc: "claim and proof session service ID doesn't match",
			proofMsgFn: func(t *testing.T) *suppliertypes.MsgSubmitProof {

				invalidSessionHeaderWrongServiceID := copy.DeepCopyJSON[sessiontypes.SessionHeader](t, *claimedSessionHeader)
				invalidSessionHeaderWrongServiceID.Service.Id = "nosvc99"

				return &suppliertypes.MsgSubmitProof{
					SupplierAddress: claim.GetSupplierAddress(),
					SessionHeader:   &invalidSessionHeaderWrongServiceID,
					Proof:           validClosestMerkleProofBz,
				}
			},
			expectedErr: status.Error(
				codes.FailedPrecondition,
				suppliertypes.ErrSupplierInvalidServiceID.Wrapf(
					"claim service ID %q does not match proof service ID %q",
					claim.GetSessionHeader().GetService().GetId(),
					"nosvc99",
				).Error(),
			),
		},
		{
			desc: "claim and proof supplier address doesn't match",
			proofMsgFn: func(t *testing.T) *suppliertypes.MsgSubmitProof {
				return &suppliertypes.MsgSubmitProof{
					SupplierAddress: sample.AccAddress(),
					SessionHeader:   claimedSessionHeader,
					Proof:           validClosestMerkleProofBz,
				}
			},
			expectedErr: suppliertypes.ErrSupplierNotFound,
		},
		{
			desc: "claim and proof session start height doesn't match",
			proofMsgFn: func(t *testing.T) *suppliertypes.MsgSubmitProof {
				invalidSessionHeaderWrongSessionStartHeight := copy.DeepCopyJSON[sessiontypes.SessionHeader](t, *claimedSessionHeader)
				invalidSessionHeaderWrongSessionStartHeight.SessionStartBlockHeight = 9999

				return &suppliertypes.MsgSubmitProof{
					SupplierAddress: claim.GetSupplierAddress(),
					SessionHeader:   &invalidSessionHeaderWrongSessionStartHeight,
					Proof:           validClosestMerkleProofBz,
				}
			},
			expectedErr: suppliertypes.ErrSupplierInvalidSessionStartHeight,
		},
		{
			desc: "claim and proof session end height doesn't match",
			proofMsgFn: func(t *testing.T) *suppliertypes.MsgSubmitProof {
				invalidSessionHeaderWrongSessionEndHeight := copy.DeepCopyJSON[sessiontypes.SessionHeader](t, *claimedSessionHeader)
				invalidSessionHeaderWrongSessionEndHeight.SessionEndBlockHeight = 9999

				return &suppliertypes.MsgSubmitProof{
					SupplierAddress: claim.GetSupplierAddress(),
					SessionHeader:   &invalidSessionHeaderWrongSessionEndHeight,
					Proof:           validClosestMerkleProofBz,
				}
			},
			expectedErr: suppliertypes.ErrSupplierInvalidSessionEndHeight,
		},
		{
			desc: "proof msg closest merkle proof is malformed",
			// Construct a valid submit proof message which has no corresponding claim.
			proofMsgFn: func(t *testing.T) *suppliertypes.MsgSubmitProof {
				// TODO_NEXT(@bryanchriswhite #141): make this test pass.
				t.SkipNow()

				// Construct an invalid proof by XORing the closest merkle proof with a
				// pseudo-random byte slice of equal length.
				malformedClosestMerkleProofBz := make([]byte, len(validClosestMerkleProofBz))
				for proofByteIdx, proofByte := range validClosestMerkleProofBz {
					// Generate a pseudo-random max 8-bit (byte-size) unsigned integer
					randByte := byte(rand.Intn(0b11111111))

					// XOR the proof byte with the pseudo-random byte
					malformedClosestMerkleProofBz[proofByteIdx] = proofByte ^ randByte
				}

				return &suppliertypes.MsgSubmitProof{
					SupplierAddress: claim.GetSupplierAddress(),
					SessionHeader:   claimedSessionHeader,
					Proof:           malformedClosestMerkleProofBz,
				}
			},
			expectedErr: suppliertypes.ErrSupplierInvalidClosestMerkleProof,
		},
		{
			desc: "proof msg closest merkle proof is invalid with wrong pseudo-random path",
			proofMsgFn: func(t *testing.T) *suppliertypes.MsgSubmitProof {
				// TODO_NEXT(@bryanchriswhite #141): make this test pass.
				t.SkipNow()

				// Generate a new valid session tree but generate a closest merkle
				// proof using the wrong path.
				_, sessionTree := newTestClaimMsgAndSessionTree(t, claimedAppSupplierPair, claimedSessionFixture)
				invalidPathClosestMerkleProof, err := sessionTree.ProveClosest(wrongTestProofPath)
				invalidPathClosestMerkleProofBz, err := invalidPathClosestMerkleProof.Marshal()
				require.NoError(t, err)

				return &suppliertypes.MsgSubmitProof{
					SupplierAddress: claim.GetSupplierAddress(),
					SessionHeader:   claimedSessionHeader,
					Proof:           invalidPathClosestMerkleProofBz,
				}
			},
			expectedErr: suppliertypes.ErrSupplierInvalidClosestMerkleProof,
		},
		{
			desc: "proof msg closest path relay application address doesn't match proof msg session application address",
			proofMsgFn: func(t *testing.T) *suppliertypes.MsgSubmitProof {
				// TODO_NEXT(@bryanchriswhite #141): make this test pass.
				t.SkipNow()
				return nil
			},
			expectedErr: suppliertypes.ErrSupplierInvalidClosestMerkleProof,
		},
		{
			desc: "invalid closest relay request application signature",
			proofMsgFn: func(t *testing.T) *suppliertypes.MsgSubmitProof {
				// TODO_NEXT(@bryanchriswhite #141): make this test pass.
				t.SkipNow()
				return nil
			},
			expectedErr: suppliertypes.ErrSupplierInvalidClosestMerkleProof,
		},
		{
			desc: "invalid closest relay response application signature",
			proofMsgFn: func(t *testing.T) *suppliertypes.MsgSubmitProof {
				// TODO_NEXT(@bryanchriswhite #141): make this test pass.
				t.SkipNow()
				return nil
			},
			expectedErr: suppliertypes.ErrSupplierInvalidClosestMerkleProof,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			submitProofRes, err := srv.SubmitProof(ctx, test.proofMsgFn(t))
			require.ErrorContains(t, err, test.expectedErr.Error())
			require.Nil(t, submitProofRes)
		})
	}
}

func ensureTestClaim(
	t *testing.T,
	sdkCtx *sdk.Context,
	supplierKeeper *keeper.Keeper,
	appSupplierPair *supplier.AppSupplierPair,
	sessionFixture *sessiontypes.Session,
) (_ *suppliertypes.Claim, closestMerkleProofBz []byte) {
	srv := keeper.NewMsgServerImpl(*supplierKeeper)
	ctx := sdk.WrapSDKContext(*sdkCtx)

	// Create a claim for the session to simulate a valid on-chain state such
	// that a valid submit proof message would be committed on-chain.
	claimMsg, sessionTree := newTestClaimMsgAndSessionTree(t, appSupplierPair, sessionFixture)
	closestMerkleProof, err := sessionTree.ProveClosest(testProofPath)
	require.NoError(t, err)

	createClaimRes, err := srv.CreateClaim(ctx, claimMsg)
	require.NoError(t, err)
	require.NotNil(t, createClaimRes)

	claimRes, err := supplierKeeper.AllClaims(sdkCtx, &suppliertypes.QueryAllClaimsRequest{})
	require.NoError(t, err)

	claims := claimRes.GetClaim()
	require.Lenf(t, claims, 1, "expected 1 claim, got %d; ensure #TestMsgServer_CreateClaim_Success() is passing", len(claims))

	claim := claims[0]
	closestMerkleProofBz, err = closestMerkleProof.Marshal()
	require.NoError(t, err)

	return &claim, closestMerkleProofBz
}

func newTestClaimMsgAndSessionTree(
	t *testing.T,
	appSupplierPair *supplier.AppSupplierPair,
	sessionFixture *sessiontypes.Session,
) (*suppliertypes.MsgCreateClaim, relayer.SessionTree) {
	t.Helper()

	tmpSmtStorePath, err := os.MkdirTemp("", t.Name())
	require.NoError(t, err)

	sessionTree, err := session.NewSessionTree(
		sessionFixture.GetHeader(),
		tmpSmtStorePath,
		noopRemoveSessionTree,
	)
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		relay := testrelayer.NewMinedRelay(t, sessionFixture.GetHeader())
		err := sessionTree.Update(relay.Hash, relay.Bytes, 1)
		require.NoError(t, err)

	}

	root, err := sessionTree.Flush()
	require.NoError(t, err)

	claimMsg := &suppliertypes.MsgCreateClaim{
		SupplierAddress: appSupplierPair.SupplierAddr,
		SessionHeader:   sessionFixture.GetHeader(),
		RootHash:        root,
	}

	return claimMsg, sessionTree
}

func noopRemoveSessionTree(*sessiontypes.SessionHeader) {}
