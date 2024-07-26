package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/depinject"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/pkg/crypto/rings"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/pkg/relayer"
	testutilevents "github.com/pokt-network/poktroll/testutil/events"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/testkeyring"
	"github.com/pokt-network/poktroll/testutil/testtree"
	"github.com/pokt-network/poktroll/x/proof/keeper"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	"github.com/pokt-network/poktroll/x/shared"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// TODO_TECHDEBT(@bryanchriswhite): Simplify this file; https://github.com/pokt-network/poktroll/pull/417#pullrequestreview-1958582600

const (
	supplierUid = "supplier"
)

var (
	blockHeaderHash         []byte
	expectedMerkleProofPath []byte

	// testProofParams sets:
	//  - the relay difficulty target hash to the easiest difficulty so that these tests don't need to mine for valid relays.
	//  - the proof request probability to 1 so that all test sessions require a proof.
	testProofParams = prooftypes.Params{
		RelayDifficultyTargetHash: protocol.BaseRelayDifficultyHashBz,
		ProofRequestProbability:   1,
	}
)

func init() {
	// The CometBFT header hash is 32 bytes: https://docs.cometbft.com/main/spec/core/data_structures
	blockHeaderHash = make([]byte, 32)
}

func TestMsgServer_SubmitProof_Success(t *testing.T) {
	tests := []struct {
		desc              string
		getProofMsgHeight func(
			sharedParams *sharedtypes.Params,
			queryHeight int64,
			supplierAddr string,
		) int64
	}{
		{
			desc: "proof message height equals supplier's earliest proof commit height",
			getProofMsgHeight: func(sharedParams *sharedtypes.Params, queryHeight int64, supplierAddr string) int64 {
				return shared.GetEarliestSupplierProofCommitHeight(
					sharedParams,
					queryHeight,
					blockHeaderHash,
					supplierAddr,
				)
			},
		},
		{
			desc: "proof message height equals proof window close height",
			getProofMsgHeight: func(sharedParams *sharedtypes.Params, queryHeight int64, _ string) int64 {
				return shared.GetProofWindowCloseHeight(sharedParams, queryHeight)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			opts := []keepertest.ProofKeepersOpt{
				// Set block hash so we can have a deterministic expected on-chain proof requested by the protocol.
				keepertest.WithBlockHash(blockHeaderHash),
				// Set block height to 1 so there is a valid session on-chain.
				keepertest.WithBlockHeight(1),
			}
			keepers, ctx := keepertest.NewProofModuleKeepers(t, opts...)
			sharedParams := keepers.SharedKeeper.GetParams(ctx)
			sdkCtx := cosmostypes.UnwrapSDKContext(ctx)

			// Set proof keeper params to disable relay mining and always require a proof.
			err := keepers.Keeper.SetParams(ctx, testProofParams)
			require.NoError(t, err)

			// Construct a keyring to hold the keypairs for the accounts used in the test.
			keyRing := keyring.NewInMemory(keepers.Codec)

			// Create a pre-generated account iterator to create accounts for the test.
			preGeneratedAccts := testkeyring.PreGeneratedAccounts()

			// Create accounts in the account keeper with corresponding keys in the
			// keyring for the application and supplier.
			supplierAddr := testkeyring.CreateOnChainAccount(
				ctx, t,
				supplierUid,
				keyRing,
				keepers,
				preGeneratedAccts,
			).String()
			appAddr := testkeyring.CreateOnChainAccount(
				ctx, t,
				"app",
				keyRing,
				keepers,
				preGeneratedAccts,
			).String()

			service := &sharedtypes.Service{Id: testServiceId}

			// Add a supplier and application pair that are expected to be in the session.
			keepers.AddServiceActors(ctx, t, service, supplierAddr, appAddr)

			// Get the session for the application/supplier pair which is expected
			// to be claimed and for which a valid proof would be accepted.
			// Given the setup above, it is guaranteed that the supplier created
			// will be part of the session.
			sessionHeader := keepers.GetSessionHeader(ctx, t, appAddr, service, 1)

			// Construct a proof message server from the proof keeper.
			srv := keeper.NewMsgServerImpl(*keepers.Keeper)

			// Prepare a ring client to sign & validate relays.
			ringClient, err := rings.NewRingClient(depinject.Supply(
				polyzero.NewLogger(),
				prooftypes.NewAppKeeperQueryClient(keepers.ApplicationKeeper),
				prooftypes.NewAccountKeeperQueryClient(keepers.AccountKeeper),
				prooftypes.NewSharedKeeperQueryClient(keepers.SharedKeeper, keepers.SessionKeeper),
			))
			require.NoError(t, err)

			// Submit the corresponding proof.
			expectedNumRelays := uint(5)
			sessionTree := testtree.NewFilledSessionTree(
				ctx, t,
				expectedNumRelays,
				supplierUid, supplierAddr,
				sessionHeader, sessionHeader, sessionHeader,
				keyRing,
				ringClient,
			)

			// Advance the block height to the test claim msg height.
			claimMsgHeight := shared.GetEarliestSupplierClaimCommitHeight(
				&sharedParams,
				sessionHeader.GetSessionEndBlockHeight(),
				blockHeaderHash,
				supplierAddr,
			)
			ctx = keepertest.SetBlockHeight(ctx, claimMsgHeight)

			// Create a valid claim.
			claim := createClaimAndStoreBlockHash(
				ctx, t, 1,
				supplierAddr,
				appAddr,
				service,
				sessionTree,
				sessionHeader,
				srv,
				keepers,
			)

			// Advance the block height to the proof path seed height.
			earliestSupplierProofCommitHeight := shared.GetEarliestSupplierProofCommitHeight(
				&sharedParams,
				sessionHeader.GetSessionEndBlockHeight(),
				blockHeaderHash,
				supplierAddr,
			)
			ctx = keepertest.SetBlockHeight(ctx, earliestSupplierProofCommitHeight-1)

			// Store proof path seed block hash in the session keeper so that it can
			// look it up during proof validation.
			keepers.StoreBlockHash(ctx)

			// Compute expected proof path.
			expectedMerkleProofPath = protocol.GetPathForProof(blockHeaderHash, sessionHeader.GetSessionId())

			// Advance the block height to the test proof msg height.
			proofMsgHeight := test.getProofMsgHeight(&sharedParams, sessionHeader.GetSessionEndBlockHeight(), supplierAddr)
			ctx = keepertest.SetBlockHeight(ctx, proofMsgHeight)

			proofMsg := newTestProofMsg(t,
				supplierAddr,
				sessionHeader,
				sessionTree,
				expectedMerkleProofPath,
			)
			submitProofRes, err := srv.SubmitProof(ctx, proofMsg)
			require.NoError(t, err)
			require.NotNil(t, submitProofRes)

			proofRes, err := keepers.AllProofs(ctx, &prooftypes.QueryAllProofsRequest{})
			require.NoError(t, err)

			proofs := proofRes.GetProofs()
			require.Lenf(t, proofs, 1, "expected 1 proof, got %d", len(proofs))
			require.Equal(t, proofMsg.SessionHeader.SessionId, proofs[0].GetSessionHeader().GetSessionId())
			require.Equal(t, proofMsg.SupplierAddress, proofs[0].GetSupplierAddress())
			require.Equal(t, proofMsg.SessionHeader.GetSessionEndBlockHeight(), proofs[0].GetSessionHeader().GetSessionEndBlockHeight())

			events := sdkCtx.EventManager().Events()
			require.Equal(t, 2, len(events))

			proofSubmittedEvents := testutilevents.FilterEvents[*prooftypes.EventProofSubmitted](t, events, "poktroll.proof.EventProofSubmitted")
			require.Equal(t, 1, len(proofSubmittedEvents))

			proofSubmittedEvent := proofSubmittedEvents[0]

			require.EqualValues(t, claim, proofSubmittedEvent.GetClaim())
			require.EqualValues(t, &proofs[0], proofSubmittedEvent.GetProof())
			require.Equal(t, uint64(expectedNumComputeUnits), proofSubmittedEvent.GetNumComputeUnits())
			require.Equal(t, uint64(expectedNumRelays), proofSubmittedEvent.GetNumRelays())
		})
	}
}

func TestMsgServer_SubmitProof_Error_OutsideOfWindow(t *testing.T) {
	var claimWindowOpenHeightBlockHash, proofWindowOpenHeightBlockHash []byte

	opts := []keepertest.ProofKeepersOpt{
		// Set block hash so we can have a deterministic expected on-chain proof requested by the protocol.
		keepertest.WithBlockHash(blockHeaderHash),
		// Set block height to 1 so there is a valid session on-chain.
		keepertest.WithBlockHeight(1),
	}
	keepers, ctx := keepertest.NewProofModuleKeepers(t, opts...)

	// Set proof keeper params to disable relaymining and always require a proof.
	err := keepers.Keeper.SetParams(ctx, testProofParams)
	require.NoError(t, err)

	// Construct a keyring to hold the keypairs for the accounts used in the test.
	keyRing := keyring.NewInMemory(keepers.Codec)

	// Create a pre-generated account iterator to create accounts for the test.
	preGeneratedAccts := testkeyring.PreGeneratedAccounts()

	// Create accounts in the account keeper with corresponding keys in the keyring for the application and supplier.
	supplierAddr := testkeyring.CreateOnChainAccount(
		ctx, t,
		supplierUid,
		keyRing,
		keepers,
		preGeneratedAccts,
	).String()
	appAddr := testkeyring.CreateOnChainAccount(
		ctx, t,
		"app",
		keyRing,
		keepers,
		preGeneratedAccts,
	).String()

	service := &sharedtypes.Service{Id: testServiceId}

	// Add a supplier and application pair that are expected to be in the session.
	keepers.AddServiceActors(ctx, t, service, supplierAddr, appAddr)

	// Get the session for the application/supplier pair which is expected
	// to be claimed and for which a valid proof would be accepted.
	// Given the setup above, it is guaranteed that the supplier created
	// will be part of the session.
	sessionHeader := keepers.GetSessionHeader(ctx, t, appAddr, service, 1)

	// Construct a proof message server from the proof keeper.
	srv := keeper.NewMsgServerImpl(*keepers.Keeper)

	// Prepare a ring client to sign & validate relays.
	ringClient, err := rings.NewRingClient(depinject.Supply(
		polyzero.NewLogger(),
		prooftypes.NewAppKeeperQueryClient(keepers.ApplicationKeeper),
		prooftypes.NewAccountKeeperQueryClient(keepers.AccountKeeper),
		prooftypes.NewSharedKeeperQueryClient(keepers.SharedKeeper, keepers.SessionKeeper),
	))
	require.NoError(t, err)

	// Submit the corresponding proof.
	numRelays := uint(5)
	sessionTree := testtree.NewFilledSessionTree(
		ctx, t,
		numRelays,
		supplierUid, supplierAddr,
		sessionHeader, sessionHeader, sessionHeader,
		keyRing,
		ringClient,
	)

	// Advance the block height to the claim window open height.
	sharedParams := keepers.SharedKeeper.GetParams(ctx)
	claimMsgHeight := shared.GetEarliestSupplierClaimCommitHeight(
		&sharedParams,
		sessionHeader.GetSessionEndBlockHeight(),
		claimWindowOpenHeightBlockHash,
		supplierAddr,
	)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(claimMsgHeight)
	ctx = sdkCtx

	// Create a valid claim.
	createClaimAndStoreBlockHash(
		ctx, t, 1,
		supplierAddr,
		appAddr,
		service,
		sessionTree,
		sessionHeader,
		srv,
		keepers,
	)

	earliestProofCommitHeight := shared.GetEarliestSupplierProofCommitHeight(
		&sharedParams,
		sessionHeader.GetSessionEndBlockHeight(),
		proofWindowOpenHeightBlockHash,
		supplierAddr,
	)
	proofWindowCloseHeight := shared.GetProofWindowCloseHeight(&sharedParams, sessionHeader.GetSessionEndBlockHeight())

	tests := []struct {
		desc           string
		proofMsgHeight int64
		expectedErr    error
	}{
		{
			desc:           "proof message height equals proof window open height minus one",
			proofMsgHeight: int64(earliestProofCommitHeight) - 1,
			expectedErr: status.Error(
				codes.FailedPrecondition,
				prooftypes.ErrProofProofOutsideOfWindow.Wrapf(
					"current block height (%d) is less than session's earliest proof commit height (%d)",
					int64(earliestProofCommitHeight)-1,
					earliestProofCommitHeight,
				).Error(),
			),
		},
		{
			desc:           "proof message height equals proof window close height plus one",
			proofMsgHeight: int64(proofWindowCloseHeight) + 1,
			expectedErr: status.Error(
				codes.FailedPrecondition,
				prooftypes.ErrProofProofOutsideOfWindow.Wrapf(
					"current block height (%d) is greater than session proof window close height (%d)",
					int64(proofWindowCloseHeight)+1,
					proofWindowCloseHeight,
				).Error(),
			),
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			// Advance the block height to the test proof msg height.
			sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
			sdkCtx = sdkCtx.WithBlockHeight(test.proofMsgHeight)
			ctx = sdkCtx

			proofMsg := newTestProofMsg(t,
				supplierAddr,
				sessionHeader,
				sessionTree,
				expectedMerkleProofPath,
			)
			_, err := srv.SubmitProof(ctx, proofMsg)
			require.ErrorContains(t, err, test.expectedErr.Error())

			proofRes, err := keepers.AllProofs(ctx, &prooftypes.QueryAllProofsRequest{})
			require.NoError(t, err)

			proofs := proofRes.GetProofs()
			require.Lenf(t, proofs, 0, "expected 0 proof, got %d", len(proofs))

			// Assert that only the create claim event was emitted.
			events := sdkCtx.EventManager().Events()
			require.Equal(t, 1, len(events))
			require.Equal(t, "poktroll.proof.EventClaimCreated", events[0].Type)
		})
	}
}

func TestMsgServer_SubmitProof_Error(t *testing.T) {
	opts := []keepertest.ProofKeepersOpt{
		// Set block hash such that on-chain closest merkle proof validation
		// uses the expected path.
		keepertest.WithBlockHash(blockHeaderHash),
		// Set block height to 1 so there is a valid session on-chain.
		keepertest.WithBlockHeight(1),
	}
	keepers, ctx := keepertest.NewProofModuleKeepers(t, opts...)

	// Ensure the minimum relay difficulty bits is set to zero so that test cases
	// don't need to mine for valid relays.
	err := keepers.Keeper.SetParams(ctx, testProofParams)
	require.NoError(t, err)

	// Construct a keyring to hold the keypairs for the accounts used in the test.
	keyRing := keyring.NewInMemory(keepers.Codec)

	// Create a pre-generated account iterator to create accounts for the test.
	preGeneratedAccts := testkeyring.PreGeneratedAccounts()

	// Create accounts in the account keeper with corresponding keys in the keyring
	// for the applications and suppliers used in the tests.
	supplierAddr := testkeyring.CreateOnChainAccount(
		ctx, t,
		supplierUid,
		keyRing,
		keepers,
		preGeneratedAccts,
	).String()
	wrongSupplierAddr := testkeyring.CreateOnChainAccount(
		ctx, t,
		"wrong_supplier",
		keyRing,
		keepers,
		preGeneratedAccts,
	).String()
	appAddr := testkeyring.CreateOnChainAccount(
		ctx, t,
		"app",
		keyRing,
		keepers,
		preGeneratedAccts,
	).String()
	wrongAppAddr := testkeyring.CreateOnChainAccount(
		ctx, t,
		"wrong_app",
		keyRing,
		keepers,
		preGeneratedAccts,
	).String()

	service := &sharedtypes.Service{Id: testServiceId}
	wrongService := &sharedtypes.Service{Id: "wrong_svc"}

	// Add a supplier and application pair that are expected to be in the session.
	keepers.AddServiceActors(ctx, t, service, supplierAddr, appAddr)

	// Add a supplier and application pair that are *not* expected to be in the session.
	keepers.AddServiceActors(ctx, t, wrongService, wrongSupplierAddr, wrongAppAddr)

	// Get the session for the application/supplier pair which is expected
	// to be claimed and for which a valid proof would be accepted.
	validSessionHeader := keepers.GetSessionHeader(ctx, t, appAddr, service, 1)

	// Construct a session header with session ID that doesn't match the expected session ID.
	wrongSessionIdHeader := *validSessionHeader
	wrongSessionIdHeader.SessionId = "wrong session ID"

	// Construct a proof message server from the proof keeper.
	srv := keeper.NewMsgServerImpl(*keepers.Keeper)

	// Construct a ringClient to get the application's ring & verify the relay
	// request signature.
	ringClient, err := rings.NewRingClient(depinject.Supply(
		polyzero.NewLogger(),
		prooftypes.NewAppKeeperQueryClient(keepers.ApplicationKeeper),
		prooftypes.NewAccountKeeperQueryClient(keepers.AccountKeeper),
		prooftypes.NewSharedKeeperQueryClient(keepers.SharedKeeper, keepers.SessionKeeper),
	))
	require.NoError(t, err)

	// Construct a valid session tree with 5 relays.
	numRelays := uint(5)
	validSessionTree := testtree.NewFilledSessionTree(
		ctx, t,
		numRelays,
		supplierUid, supplierAddr,
		validSessionHeader, validSessionHeader, validSessionHeader,
		keyRing,
		ringClient,
	)

	// Advance the block height to the earliest claim commit height.
	sharedParams := keepers.SharedKeeper.GetParams(ctx)
	claimMsgHeight := shared.GetEarliestSupplierClaimCommitHeight(
		&sharedParams,
		validSessionHeader.GetSessionEndBlockHeight(),
		blockHeaderHash,
		supplierAddr,
	)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(claimMsgHeight)
	ctx = sdkCtx

	// Create a valid claim for the expected session and update the block hash
	// store for the corresponding session.
	createClaimAndStoreBlockHash(
		ctx, t, 1,
		supplierAddr,
		appAddr,
		service,
		validSessionTree,
		validSessionHeader,
		srv,
		keepers,
	)

	tests := []struct {
		desc        string
		newProofMsg func(t *testing.T) *prooftypes.MsgSubmitProof
		expectedErr error
	}{
		{
			desc: "proof service ID cannot be empty",
			newProofMsg: func(t *testing.T) *prooftypes.MsgSubmitProof {
				// Set proof session ID to empty string.
				emptySessionIdHeader := *validSessionHeader
				emptySessionIdHeader.SessionId = ""

				// Construct new proof message.
				return newTestProofMsg(t,
					supplierAddr,
					&emptySessionIdHeader,
					validSessionTree,
					expectedMerkleProofPath,
				)
			},
			expectedErr: status.Error(
				codes.InvalidArgument,
				prooftypes.ErrProofInvalidSessionId.Wrapf(
					"session ID does not match on-chain session ID; expected %q, got %q",
					validSessionHeader.GetSessionId(),
					"",
				).Error(),
			),
		},
		{
			desc: "merkle proof cannot be empty",
			newProofMsg: func(t *testing.T) *prooftypes.MsgSubmitProof {
				// Construct new proof message.
				proof := newTestProofMsg(t,
					supplierAddr,
					validSessionHeader,
					validSessionTree,
					expectedMerkleProofPath,
				)

				// Set merkle proof to an empty byte slice.
				proof.Proof = []byte{}
				return proof
			},
			expectedErr: status.Error(
				codes.InvalidArgument,
				prooftypes.ErrProofInvalidProof.Wrap(
					"proof cannot be empty",
				).Error(),
			),
		},
		{
			desc: "proof session ID must match on-chain session ID",
			newProofMsg: func(t *testing.T) *prooftypes.MsgSubmitProof {
				// Construct new proof message using the wrong session ID.
				return newTestProofMsg(t,
					supplierAddr,
					&wrongSessionIdHeader,
					validSessionTree,
					expectedMerkleProofPath,
				)
			},
			expectedErr: status.Error(
				codes.InvalidArgument,
				prooftypes.ErrProofInvalidSessionId.Wrapf(
					"session ID does not match on-chain session ID; expected %q, got %q",
					validSessionHeader.GetSessionId(),
					wrongSessionIdHeader.GetSessionId(),
				).Error(),
			),
		},
		{
			desc: "proof supplier must be in on-chain session",
			newProofMsg: func(t *testing.T) *prooftypes.MsgSubmitProof {
				// Construct a proof message with a  supplier that does not belong in the session.
				return newTestProofMsg(t,
					wrongSupplierAddr,
					validSessionHeader,
					validSessionTree,
					expectedMerkleProofPath,
				)
			},
			expectedErr: status.Error(
				codes.InvalidArgument,
				prooftypes.ErrProofNotFound.Wrapf(
					"supplier address %q not found in session ID %q",
					wrongSupplierAddr,
					validSessionHeader.GetSessionId(),
				).Error(),
			),
		},
	}

	// Submit the corresponding proof.
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			proofMsg := test.newProofMsg(t)

			// Advance the block height to the proof path seed height.
			earliestSupplierProofCommitHeight := shared.GetEarliestSupplierProofCommitHeight(
				&sharedParams,
				proofMsg.GetSessionHeader().GetSessionEndBlockHeight(),
				blockHeaderHash,
				proofMsg.GetSupplierAddress(),
			)
			ctx = keepertest.SetBlockHeight(ctx, earliestSupplierProofCommitHeight-1)

			// Store proof path seed block hash in the session keeper so that it can
			// look it up during proof validation.
			keepers.StoreBlockHash(ctx)

			// Advance the block height to the earliest proof commit height.
			ctx = keepertest.SetBlockHeight(ctx, earliestSupplierProofCommitHeight)

			submitProofRes, err := srv.SubmitProof(ctx, proofMsg)

			require.ErrorContains(t, err, test.expectedErr.Error())
			require.Nil(t, submitProofRes)

			proofRes, err := keepers.AllProofs(ctx, &prooftypes.QueryAllProofsRequest{})
			require.NoError(t, err)

			// Expect zero proofs to have been persisted as all test cases are error cases.
			proofs := proofRes.GetProofs()
			require.Lenf(t, proofs, 0, "expected 0 proofs, got %d", len(proofs))

			// Assert that no proof submitted events were emitted.
			events := sdkCtx.EventManager().Events()
			proofSubmittedEvents := testutilevents.FilterEvents[*prooftypes.EventProofSubmitted](t, events, "poktroll.proof.EventProofSubmitted")
			require.Equal(t, 0, len(proofSubmittedEvents))
		})
	}
}

// newTestProofMsg creates a new submit proof message that can be submitted
// to be validated and stored on-chain.
func newTestProofMsg(
	t *testing.T,
	supplierAddr string,
	sessionHeader *sessiontypes.SessionHeader,
	sessionTree relayer.SessionTree,
	closestProofPath []byte,
) *prooftypes.MsgSubmitProof {
	t.Helper()

	// Generate a closest proof from the session tree using closestProofPath.
	merkleProof, err := sessionTree.ProveClosest(closestProofPath)
	require.NoError(t, err)
	require.NotNil(t, merkleProof)

	// Serialize the closest merkle proof.
	merkleProofBz, err := merkleProof.Marshal()
	require.NoError(t, err)

	return &prooftypes.MsgSubmitProof{
		SupplierAddress: supplierAddr,
		SessionHeader:   sessionHeader,
		Proof:           merkleProofBz,
	}
}

// createClaimAndStoreBlockHash creates a valid claim, submits it on-chain,
// and on success, stores the block hash for retrieval at future heights.
// TODO_TECHDEBT(@bryanchriswhite): Consider if we could/should split
// this into two functions.
func createClaimAndStoreBlockHash(
	ctx context.Context,
	t *testing.T,
	sessionStartHeight int64,
	supplierAddr, appAddr string,
	service *sharedtypes.Service,
	sessionTree relayer.SessionTree,
	sessionHeader *sessiontypes.SessionHeader,
	msgServer prooftypes.MsgServer,
	keepers *keepertest.ProofModuleKeepers,
) *prooftypes.Claim {
	merkleRootBz, err := sessionTree.Flush()
	require.NoError(t, err)

	// Create a create claim message.
	claimMsg := newTestClaimMsg(t,
		sessionStartHeight,
		sessionHeader.GetSessionId(),
		supplierAddr,
		appAddr,
		service,
		merkleRootBz,
	)
	claimRes, err := msgServer.CreateClaim(ctx, claimMsg)
	require.NoError(t, err)

	sharedParams := keepers.SharedKeeper.GetParams(ctx)

	claimWindowOpenHeight := shared.GetClaimWindowOpenHeight(
		&sharedParams,
		sessionStartHeight,
	)

	ctx = keepertest.SetBlockHeight(ctx, claimWindowOpenHeight)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)

	earliestSupplierClaimCommitHeight := shared.GetEarliestSupplierClaimCommitHeight(
		&sharedParams,
		sessionStartHeight,
		sdkCtx.HeaderHash(),
		supplierAddr,
	)

	// Set block height to be after the session grace period.
	earliestSupplierClaimCommitCtx := keepertest.SetBlockHeight(ctx, earliestSupplierClaimCommitHeight)

	// Store the current context's block hash for future height, which is currently an EndBlocker operation.
	keepers.StoreBlockHash(earliestSupplierClaimCommitCtx)

	return claimRes.GetClaim()
}
