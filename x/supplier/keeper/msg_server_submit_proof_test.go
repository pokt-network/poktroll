package keeper_test

import (
	"math/rand"
	"os"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
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
	sessionFixtures := supplier.NewSessionFixturesWithPairings(t, service, appSupplierPair)
	sessionFixture := sessionFixtures.GetSession(t, appSupplierPair)

	supplierKeeper, sdkCtx := keepertest.SupplierKeeper(t, sessionFixtures)
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

func TestMsgServer_SubmitProof_ErrorSessionHeaderValidation(t *testing.T) {
	testService := &sharedtypes.Service{Id: testServiceId}
	claimedAppSupplierPair := &supplier.AppSupplierPair{
		AppAddr:      sample.AccAddress(),
		SupplierAddr: sample.AccAddress(),
	}
	unClaimedAppSupplierPair := &supplier.AppSupplierPair{
		AppAddr:      sample.AccAddress(),
		SupplierAddr: claimedAppSupplierPair.SupplierAddr,
	}
	sessionFixtures := supplier.NewSessionFixturesWithPairings(
		t, testService,
		claimedAppSupplierPair,
		unClaimedAppSupplierPair,
	)

	supplierKeeper, sdkCtx := keepertest.SupplierKeeper(t, sessionFixtures)
	claimedSessionFixture := sessionFixtures.GetSession(t, claimedAppSupplierPair)
	claimedSessionHeader := claimedSessionFixture.GetHeader()
	claim, claimedClosestMerkleProofBz := ensureTestClaim(
		t, &sdkCtx,
		supplierKeeper,
		claimedAppSupplierPair,
		claimedSessionFixture,
	)
	require.Equal(t, claimedAppSupplierPair.SupplierAddr, claim.GetSupplierAddress())

	srv := keeper.NewMsgServerImpl(*supplierKeeper)
	ctx := sdk.WrapSDKContext(sdkCtx)

	randSupplierAddr := sample.AccAddress()

	tests := []struct {
		desc        string
		proofMsgFn  func(t *testing.T) *suppliertypes.MsgSubmitProof
		expectedErr error
	}{
		{
			desc: "proof msg session ID doesn't match on-chain session ID",
			// Construct a proof message by deriving a session header from the
			// claimed session header and changing the session ID.
			proofMsgFn: func(t *testing.T) *suppliertypes.MsgSubmitProof {
				invalidSessionHeaderWrongSessionID := copy.DeepCopyJSON[sessiontypes.SessionHeader](t, *claimedSessionHeader)
				invalidSessionHeaderWrongSessionID.SessionId = "wrong_session_id"

				return &suppliertypes.MsgSubmitProof{
					SupplierAddress: claim.GetSupplierAddress(),
					SessionHeader:   &invalidSessionHeaderWrongSessionID,
					Proof:           claimedClosestMerkleProofBz,
				}
			},
			expectedErr: suppliertypes.ErrSupplierInvalidSessionId,
		},
		{
			// Construct a proof message by deriving a session header from the
			// claimed session header and changing the service ID.
			desc: "proof msg session service ID doesn't match on-chain session",
			proofMsgFn: func(t *testing.T) *suppliertypes.MsgSubmitProof {
				invalidSessionHeaderWrongServiceID := copy.DeepCopyJSON[sessiontypes.SessionHeader](t, *claimedSessionHeader)
				invalidSessionHeaderWrongServiceID.Service.Id = "nosvc99"

				return &suppliertypes.MsgSubmitProof{
					SupplierAddress: claim.GetSupplierAddress(),
					SessionHeader:   &invalidSessionHeaderWrongServiceID,
					Proof:           claimedClosestMerkleProofBz,
				}
			},
			// TODO_IN_THIS_COMMIT: add comment corresponding to mock eeper comment re: assertions agains this error.
			expectedErr: status.Error(codes.NotFound, suppliertypes.ErrSupplierInvalidSessionId.Error()),
		},
		{
			// Construct a proof message by using the same session header as the
			// claimed session header and a different supplier addresss than that
			// of the claimed session.
			desc: "proof msg supplier address not in the on-chain session",
			proofMsgFn: func(t *testing.T) *suppliertypes.MsgSubmitProof {
				return &suppliertypes.MsgSubmitProof{
					SupplierAddress: randSupplierAddr,
					SessionHeader:   claimedSessionHeader,
					Proof:           claimedClosestMerkleProofBz,
				}
			},
			expectedErr: status.Error(
				codes.InvalidArgument,
				suppliertypes.ErrSupplierNotFound.Wrapf(
					"supplier address %q not found in session ID %q",
					randSupplierAddr,
					sessionFixtures.GetMockSessionId(claimedAppSupplierPair),
				).Error(),
			),
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

func TestMsgServer_SubmitProof_ErrorClaimValidation(t *testing.T) {
	testService := &sharedtypes.Service{Id: testServiceId}
	claimedAppSupplierPair := &supplier.AppSupplierPair{
		AppAddr:      sample.AccAddress(),
		SupplierAddr: sample.AccAddress(),
	}
	wrongClaimedAppSupplierPair := &supplier.AppSupplierPair{
		AppAddr:      claimedAppSupplierPair.AppAddr,
		SupplierAddr: sample.AccAddress(),
	}
	unClaimedAppSupplierPair := &supplier.AppSupplierPair{
		AppAddr:      sample.AccAddress(),
		SupplierAddr: claimedAppSupplierPair.SupplierAddr,
	}
	sessionFixtures := supplier.NewSessionFixturesWithPairings(
		t, testService,
		claimedAppSupplierPair,
		wrongClaimedAppSupplierPair,
		unClaimedAppSupplierPair,
	)
	claimedSessionFixture := sessionFixtures.GetSession(t, claimedAppSupplierPair)
	claimedSessionHeader := claimedSessionFixture.GetHeader()
	unClaimedSessionFixture := sessionFixtures.GetSession(t, unClaimedAppSupplierPair)
	unClaimedSessionId := sessionFixtures.GetMockSessionId(unClaimedAppSupplierPair)

	supplierKeeper, sdkCtx := keepertest.SupplierKeeper(t, sessionFixtures)
	claim, claimedClosestMerkleProofBz := ensureTestClaim(
		t, &sdkCtx,
		supplierKeeper,
		claimedAppSupplierPair,
		claimedSessionFixture,
	)
	require.Equal(t, claimedAppSupplierPair.SupplierAddr, claim.GetSupplierAddress())

	srv := keeper.NewMsgServerImpl(*supplierKeeper)
	ctx := sdk.WrapSDKContext(sdkCtx)

	wrongSupplierAddr := sample.AccAddress()
	invalidSessionHeaderWrongSessionID := copy.DeepCopyJSON[sessiontypes.SessionHeader](t, *claimedSessionHeader)
	invalidSessionHeaderWrongSessionID.SessionId = "wrong_session_id"

	// NB: all test scenarios use valid session IDs (i.e. there is a supplier/app
	// pair staked for the same service).
	tests := []struct {
		desc        string
		proofMsgFn  func(t *testing.T) *suppliertypes.MsgSubmitProof
		expectedErr error
	}{
		{
			desc: "claim not found for proof msg session",
			proofMsgFn: func(t *testing.T) *suppliertypes.MsgSubmitProof {
				return &suppliertypes.MsgSubmitProof{
					SupplierAddress: claim.GetSupplierAddress(),
					SessionHeader:   &invalidSessionHeaderWrongSessionID,
					Proof:           claimedClosestMerkleProofBz,
				}
			},
			// TODO_IN_THIS_COMMIT: double-check that this  is the correct error to assert against. Search for "claim not found".
			expectedErr: status.Error(
				codes.InvalidArgument,
				suppliertypes.ErrSupplierInvalidSessionId.Wrapf(
					"session ID does not match on-chain session ID; expected %q, got %q",
					claimedSessionHeader.GetSessionId(),
					invalidSessionHeaderWrongSessionID.GetSessionId(),
				).Error(),
			),
		},
		{
			desc: "claim and proof session application address doesn't match",
			proofMsgFn: func(t *testing.T) *suppliertypes.MsgSubmitProof {
				return &suppliertypes.MsgSubmitProof{
					SupplierAddress: claim.GetSupplierAddress(),
					SessionHeader:   unClaimedSessionFixture.GetHeader(),
					Proof:           claimedClosestMerkleProofBz,
				}
			},
			expectedErr: status.Error(
				codes.FailedPrecondition,
				suppliertypes.ErrSupplierClaimNotFound.Wrapf(
					"no claim found for session ID %q",
					unClaimedSessionId,
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
					Proof:           claimedClosestMerkleProofBz,
				}
			},
			// TODO_IN_THIS_COMMIT: add comment corresponding to mock eeper comment re: assertions agains this error.
			expectedErr: status.Error(codes.NotFound, suppliertypes.ErrSupplierInvalidSessionId.Error()),
		},
		{
			desc: "claim and proof supplier address doesn't match",
			proofMsgFn: func(t *testing.T) *suppliertypes.MsgSubmitProof {
				t.Logf("claimedSessionHeader.GetSessionId(): %s", claimedSessionHeader.GetSessionId())
				t.Logf("unClaimedSessionHeader.GetSessionId(): %s", unClaimedSessionId)
				return &suppliertypes.MsgSubmitProof{
					SupplierAddress: wrongSupplierAddr,
					SessionHeader:   claimedSessionHeader,
					Proof:           claimedClosestMerkleProofBz,
				}
			},
			expectedErr: status.Error(
				codes.InvalidArgument,
				suppliertypes.ErrSupplierNotFound.Wrapf(
					"supplier address %q not found in session ID %q",
					wrongSupplierAddr,
					claimedSessionHeader.GetSessionId(),
				).Error(),
			),
		},
		{
			desc: "claim and proof session start height doesn't match",
			proofMsgFn: func(t *testing.T) *suppliertypes.MsgSubmitProof {
				invalidSessionHeaderWrongSessionStartHeight := copy.DeepCopyJSON[sessiontypes.SessionHeader](t, *claimedSessionHeader)
				invalidSessionHeaderWrongSessionStartHeight.SessionStartBlockHeight = 9999

				return &suppliertypes.MsgSubmitProof{
					SupplierAddress: claim.GetSupplierAddress(),
					SessionHeader:   &invalidSessionHeaderWrongSessionStartHeight,
					Proof:           claimedClosestMerkleProofBz,
				}
			},
			// TODO_IN_THIS_COMMIT: expect gRPC status error with msg.
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
					Proof:           claimedClosestMerkleProofBz,
				}
			},
			// TODO_IN_THIS_COMMIT: expect gRPC status error with msg.
			expectedErr: suppliertypes.ErrSupplierInvalidSessionEndHeight,
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

func TestMsgServer_SubmitProof_ErrorClosestMerkleProofValidation(t *testing.T) {
	// TODO_NEXT(@bryanchriswhite #141): make this test pass.
	t.SkipNow()

	testService := &sharedtypes.Service{Id: testServiceId}
	claimedAppSupplierPair := &supplier.AppSupplierPair{
		AppAddr:      sample.AccAddress(),
		SupplierAddr: sample.AccAddress(),
	}
	sessionFixtures := supplier.NewSessionFixturesWithPairings(
		t, testService,
		claimedAppSupplierPair,
	)
	claimedSessionFixture := sessionFixtures.GetSession(t, claimedAppSupplierPair)
	claimedSessionHeader := claimedSessionFixture.GetHeader()

	supplierKeeper, sdkCtx := keepertest.SupplierKeeper(t, sessionFixtures)
	claim, claimedClosestMerkleProofBz := ensureTestClaim(
		t, &sdkCtx,
		supplierKeeper,
		claimedAppSupplierPair,
		claimedSessionFixture,
	)
	require.Equal(t, claimedAppSupplierPair.SupplierAddr, claim.GetSupplierAddress())

	srv := keeper.NewMsgServerImpl(*supplierKeeper)
	ctx := sdk.WrapSDKContext(sdkCtx)

	tests := []struct {
		desc        string
		proofMsgFn  func(t *testing.T) *suppliertypes.MsgSubmitProof
		expectedErr error
	}{
		{
			desc: "proof msg closest merkle proof is malformed",
			// Construct a valid submit proof message which has no corresponding claim.
			proofMsgFn: func(t *testing.T) *suppliertypes.MsgSubmitProof {

				// Construct an invalid proof by XORing the closest merkle proof with a
				// pseudo-random byte slice of equal length.
				malformedClosestMerkleProofBz := make([]byte, len(claimedClosestMerkleProofBz))
				for proofByteIdx, proofByte := range claimedClosestMerkleProofBz {
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
			// TODO_IN_THIS_COMMIT: expect gRPC status error with msg.
			expectedErr: suppliertypes.ErrSupplierInvalidClosestMerkleProof,
		},
		{
			desc: "proof msg closest merkle proof is invalid with wrong pseudo-random path",
			proofMsgFn: func(t *testing.T) *suppliertypes.MsgSubmitProof {
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
		},
		{
			desc: "proof msg closest path relay application address doesn't match proof msg session application address",
			proofMsgFn: func(t *testing.T) *suppliertypes.MsgSubmitProof {
				return nil
			},
		},
		{
			desc: "invalid closest relay request application signature",
			proofMsgFn: func(t *testing.T) *suppliertypes.MsgSubmitProof {
				return nil
			},
		},
		{
			desc: "invalid closest relay response application signature",
			proofMsgFn: func(t *testing.T) *suppliertypes.MsgSubmitProof {
				return nil
			},
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

	claim, found := supplierKeeper.GetClaim(
		*sdkCtx,
		sessionFixture.GetSessionId(),
		appSupplierPair.SupplierAddr,
	)
	require.Truef(
		t, found,
		"expected claim to be found for session ID %q and supplier address %q",
		sessionFixture.GetSessionId(),
		appSupplierPair.SupplierAddr,
	)

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
