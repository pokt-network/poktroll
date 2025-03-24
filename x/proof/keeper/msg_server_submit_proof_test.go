package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/depinject"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/pocket/app/volatile"
	"github.com/pokt-network/pocket/pkg/crypto/protocol"
	"github.com/pokt-network/pocket/pkg/crypto/rings"
	"github.com/pokt-network/pocket/pkg/polylog/polyzero"
	"github.com/pokt-network/pocket/pkg/relayer"
	testutilevents "github.com/pokt-network/pocket/testutil/events"
	keepertest "github.com/pokt-network/pocket/testutil/keeper"
	"github.com/pokt-network/pocket/testutil/sample"
	"github.com/pokt-network/pocket/testutil/testkeyring"
	"github.com/pokt-network/pocket/testutil/testtree"
	"github.com/pokt-network/pocket/x/proof/keeper"
	prooftypes "github.com/pokt-network/pocket/x/proof/types"
	servicekeeper "github.com/pokt-network/pocket/x/service/keeper"
	sessiontypes "github.com/pokt-network/pocket/x/session/types"
	sharedtypes "github.com/pokt-network/pocket/x/shared/types"
	suppliertypes "github.com/pokt-network/pocket/x/supplier/types"
)

// TODO_TECHDEBT(@bryanchriswhite): Simplify this file; https://github.com/pokt-network/pocket/pull/417#pullrequestreview-1958582600

const (
	supplierOperatorUid = "supplier"
)

var (
	blockHeaderHash         []byte
	expectedMerkleProofPath []byte = make([]byte, protocol.TrieHasherSize)

	// testProofParams sets:
	//  - the relay difficulty target hash to the easiest difficulty so that these tests don't need to mine for valid relays.
	//  - the proof request probability to 1 so that all test sessions require a proof.
	testProofParams = prooftypes.Params{
		ProofRequestProbability: 1,
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
			supplierOperatorAddr string,
		) int64
	}{
		{
			desc: "proof message height equals supplier's earliest proof commit height",
			getProofMsgHeight: func(sharedParams *sharedtypes.Params, queryHeight int64, supplierOperatorAddr string) int64 {
				return sharedtypes.GetEarliestSupplierProofCommitHeight(
					sharedParams,
					queryHeight,
					blockHeaderHash,
					supplierOperatorAddr,
				)
			},
		},
		{
			desc: "proof message height equals proof window close height",
			getProofMsgHeight: func(sharedParams *sharedtypes.Params, queryHeight int64, _ string) int64 {
				return sharedtypes.GetProofWindowCloseHeight(sharedParams, queryHeight)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			opts := []keepertest.ProofKeepersOpt{
				// Set block hash so we can have a deterministic expected onchain proof requested by the protocol.
				keepertest.WithBlockHash(blockHeaderHash),
				// Set block height to 1 so there is a valid session onchain.
				keepertest.WithBlockHeight(1),
			}
			keepers, ctx := keepertest.NewProofModuleKeepers(t, opts...)
			sharedParams := keepers.SharedKeeper.GetParams(ctx)
			sdkCtx := cosmostypes.UnwrapSDKContext(ctx)

			// Set proof keeper params to disable relay mining and always require a proof.
			proofParams := keepers.Keeper.GetParams(ctx)
			proofParams.ProofRequestProbability = testProofParams.ProofRequestProbability
			err := keepers.Keeper.SetParams(ctx, proofParams)
			require.NoError(t, err)

			// Construct a keyring to hold the keypairs for the accounts used in the test.
			keyRing := keyring.NewInMemory(keepers.Codec)

			// Create a pre-generated account iterator to create accounts for the test.
			preGeneratedAccts := testkeyring.PreGeneratedAccounts()

			// Create accounts in the account keeper with corresponding keys in the
			// keyring for the application and supplier.
			supplierOperatorAddr := testkeyring.CreateOnChainAccount(
				ctx, t,
				supplierOperatorUid,
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

			fundSupplierOperatorAccount(t, ctx, keepers, supplierOperatorAddr)

			service := &sharedtypes.Service{
				Id:                   testServiceId,
				ComputeUnitsPerRelay: computeUnitsPerRelay,
				OwnerAddress:         sample.AccAddress(),
			}

			// Add a supplier and application pair that are expected to be in the session.
			keepers.AddServiceActors(ctx, t, service, supplierOperatorAddr, appAddr)

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
			numRelays := uint64(5)
			numClaimComputeUnits := numRelays * service.ComputeUnitsPerRelay
			sessionTree := testtree.NewFilledSessionTree(
				ctx, t,
				numRelays, service.ComputeUnitsPerRelay,
				supplierOperatorUid, supplierOperatorAddr,
				sessionHeader, sessionHeader, sessionHeader,
				keyRing,
				ringClient,
			)

			// Advance the block height to the test claim msg height.
			claimMsgHeight := sharedtypes.GetEarliestSupplierClaimCommitHeight(
				&sharedParams,
				sessionHeader.GetSessionEndBlockHeight(),
				blockHeaderHash,
				supplierOperatorAddr,
			)
			ctx = keepertest.SetBlockHeight(ctx, claimMsgHeight)

			// Create a valid claim.
			claim := createClaimAndStoreBlockHash(
				ctx, t, 1,
				supplierOperatorAddr,
				appAddr,
				service,
				sessionTree,
				sessionHeader,
				srv,
				keepers,
			)

			// Advance the block height to the proof path seed height.
			earliestSupplierProofCommitHeight := sharedtypes.GetEarliestSupplierProofCommitHeight(
				&sharedParams,
				sessionHeader.GetSessionEndBlockHeight(),
				blockHeaderHash,
				supplierOperatorAddr,
			)
			ctx = keepertest.SetBlockHeight(ctx, earliestSupplierProofCommitHeight-1)

			// Store proof path seed block hash in the session keeper so that it can
			// look it up during proof validation.
			keepers.StoreBlockHash(ctx)

			// Compute expected proof path.
			expectedMerkleProofPath = protocol.GetPathForProof(blockHeaderHash, sessionHeader.GetSessionId())

			// Advance the block height to the test proof msg height.
			proofMsgHeight := test.getProofMsgHeight(&sharedParams, sessionHeader.GetSessionEndBlockHeight(), supplierOperatorAddr)
			ctx = keepertest.SetBlockHeight(ctx, proofMsgHeight)

			proofMsg := newTestProofMsg(t,
				supplierOperatorAddr,
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
			require.Equal(t, proofMsg.SupplierOperatorAddress, proofs[0].GetSupplierOperatorAddress())
			require.Equal(t, proofMsg.SessionHeader.GetSessionEndBlockHeight(), proofs[0].GetSessionHeader().GetSessionEndBlockHeight())

			events := sdkCtx.EventManager().Events()

			claimCreatedEvents := testutilevents.FilterEvents[*prooftypes.EventClaimCreated](t, events)
			require.Len(t, claimCreatedEvents, 1)

			proofSubmittedEvents := testutilevents.FilterEvents[*prooftypes.EventProofSubmitted](t, events)
			require.Len(t, proofSubmittedEvents, 1)

			proofSubmittedEvent := proofSubmittedEvents[0]

			targetNumRelays := keepers.ServiceKeeper.GetParams(ctx).TargetNumRelays
			relayMiningDifficulty := servicekeeper.NewDefaultRelayMiningDifficulty(
				ctx,
				keepers.Logger(),
				service.Id,
				targetNumRelays,
				targetNumRelays,
			)

			numEstimatedComputUnits, err := claim.GetNumEstimatedComputeUnits(relayMiningDifficulty)
			require.NoError(t, err)

			claimedUPOKT, err := claim.GetClaimeduPOKT(sharedParams, relayMiningDifficulty)
			require.NoError(t, err)

			require.EqualValues(t, claim, proofSubmittedEvent.GetClaim())
			require.EqualValues(t, &proofs[0], proofSubmittedEvent.GetProof())
			require.Equal(t, uint64(numRelays), proofSubmittedEvent.GetNumRelays())
			require.Equal(t, uint64(numClaimComputeUnits), proofSubmittedEvent.GetNumClaimedComputeUnits())
			require.Equal(t, numEstimatedComputUnits, proofSubmittedEvent.GetNumEstimatedComputeUnits())
			require.Equal(t, &claimedUPOKT, proofSubmittedEvent.GetClaimedUpokt())
		})
	}
}

func TestMsgServer_SubmitProof_Error_OutsideOfWindow(t *testing.T) {
	var claimWindowOpenHeightBlockHash, proofWindowOpenHeightBlockHash []byte

	opts := []keepertest.ProofKeepersOpt{
		// Set block hash so we can have a deterministic expected onchain proof requested by the protocol.
		keepertest.WithBlockHash(blockHeaderHash),
		// Set block height to 1 so there is a valid session onchain.
		keepertest.WithBlockHeight(1),
	}
	keepers, ctx := keepertest.NewProofModuleKeepers(t, opts...)

	// Set proof keeper params to disable relaymining and always require a proof.
	proofParams := keepers.Keeper.GetParams(ctx)
	proofParams.ProofRequestProbability = testProofParams.ProofRequestProbability
	err := keepers.Keeper.SetParams(ctx, proofParams)
	require.NoError(t, err)

	// Construct a keyring to hold the keypairs for the accounts used in the test.
	keyRing := keyring.NewInMemory(keepers.Codec)

	// Create a pre-generated account iterator to create accounts for the test.
	preGeneratedAccts := testkeyring.PreGeneratedAccounts()

	// Create accounts in the account keeper with corresponding keys in the keyring for the application and supplier.
	supplierOperatorAddr := testkeyring.CreateOnChainAccount(
		ctx, t,
		supplierOperatorUid,
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

	service := &sharedtypes.Service{
		Id:                   testServiceId,
		ComputeUnitsPerRelay: computeUnitsPerRelay,
		OwnerAddress:         sample.AccAddress(),
	}

	fundSupplierOperatorAccount(t, ctx, keepers, supplierOperatorAddr)

	// Add a supplier and application pair that are expected to be in the session.
	keepers.AddServiceActors(ctx, t, service, supplierOperatorAddr, appAddr)

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
	numRelays := uint64(5)
	sessionTree := testtree.NewFilledSessionTree(
		ctx, t,
		numRelays, service.ComputeUnitsPerRelay,
		supplierOperatorUid, supplierOperatorAddr,
		sessionHeader, sessionHeader, sessionHeader,
		keyRing,
		ringClient,
	)

	// Advance the block height to the claim window open height.
	sharedParams := keepers.SharedKeeper.GetParams(ctx)
	claimMsgHeight := sharedtypes.GetEarliestSupplierClaimCommitHeight(
		&sharedParams,
		sessionHeader.GetSessionEndBlockHeight(),
		claimWindowOpenHeightBlockHash,
		supplierOperatorAddr,
	)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(claimMsgHeight)
	ctx = sdkCtx

	// Create a valid claim.
	createClaimAndStoreBlockHash(
		ctx, t, 1,
		supplierOperatorAddr,
		appAddr,
		service,
		sessionTree,
		sessionHeader,
		srv,
		keepers,
	)

	earliestProofCommitHeight := sharedtypes.GetEarliestSupplierProofCommitHeight(
		&sharedParams,
		sessionHeader.GetSessionEndBlockHeight(),
		proofWindowOpenHeightBlockHash,
		supplierOperatorAddr,
	)
	proofWindowCloseHeight := sharedtypes.GetProofWindowCloseHeight(&sharedParams, sessionHeader.GetSessionEndBlockHeight())

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
				supplierOperatorAddr,
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

			claimCreatedEvents := testutilevents.FilterEvents[*prooftypes.EventClaimCreated](t, events)
			require.Len(t, claimCreatedEvents, 1)

			proofSubmittedEvents := testutilevents.FilterEvents[*prooftypes.EventProofSubmitted](t, events)
			require.Len(t, proofSubmittedEvents, 0)
		})
	}
}

func TestMsgServer_SubmitProof_Error(t *testing.T) {
	opts := []keepertest.ProofKeepersOpt{
		// Set block hash such that onchain closest merkle proof validation
		// uses the expected path.
		keepertest.WithBlockHash(blockHeaderHash),
		// Set block height to 1 so there is a valid session onchain.
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
	supplierOperatorAddr := testkeyring.CreateOnChainAccount(
		ctx, t,
		supplierOperatorUid,
		keyRing,
		keepers,
		preGeneratedAccts,
	).String()
	wrongSupplierOperatorAddr := testkeyring.CreateOnChainAccount(
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

	service := &sharedtypes.Service{
		Id:                   testServiceId,
		ComputeUnitsPerRelay: computeUnitsPerRelay,
		OwnerAddress:         sample.AccAddress(),
	}
	wrongService := &sharedtypes.Service{
		Id:                   "wrong_svc",
		ComputeUnitsPerRelay: computeUnitsPerRelay,
		OwnerAddress:         sample.AccAddress(),
	}

	// Add a supplier and application pair that are expected to be in the session.
	keepers.AddServiceActors(ctx, t, service, supplierOperatorAddr, appAddr)

	// Add a supplier and application pair that are *not* expected to be in the session.
	keepers.AddServiceActors(ctx, t, wrongService, wrongSupplierOperatorAddr, wrongAppAddr)

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
	numRelays := uint64(5)
	validSessionTree := testtree.NewFilledSessionTree(
		ctx, t,
		numRelays, service.ComputeUnitsPerRelay,
		supplierOperatorUid, supplierOperatorAddr,
		validSessionHeader, validSessionHeader, validSessionHeader,
		keyRing,
		ringClient,
	)

	// Advance the block height to the earliest claim commit height.
	sharedParams := keepers.SharedKeeper.GetParams(ctx)
	claimMsgHeight := sharedtypes.GetEarliestSupplierClaimCommitHeight(
		&sharedParams,
		validSessionHeader.GetSessionEndBlockHeight(),
		blockHeaderHash,
		supplierOperatorAddr,
	)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(claimMsgHeight)
	ctx = sdkCtx

	// Create a valid claim for the expected session and update the block hash
	// store for the corresponding session.
	createClaimAndStoreBlockHash(
		ctx, t, 1,
		supplierOperatorAddr,
		appAddr,
		service,
		validSessionTree,
		validSessionHeader,
		srv,
		keepers,
	)

	tests := []struct {
		desc                            string
		newProofMsg                     func(t *testing.T) *prooftypes.MsgSubmitProof
		msgSubmitProofToExpectedErrorFn func(*prooftypes.MsgSubmitProof) error
	}{
		{
			desc: "proof service ID cannot be empty",
			newProofMsg: func(t *testing.T) *prooftypes.MsgSubmitProof {
				// Set proof session ID to empty string.
				emptySessionIdHeader := *validSessionHeader
				emptySessionIdHeader.SessionId = ""

				// Construct new proof message.
				return newTestProofMsg(t,
					supplierOperatorAddr,
					&emptySessionIdHeader,
					validSessionTree,
					expectedMerkleProofPath,
				)
			},
			msgSubmitProofToExpectedErrorFn: func(msgSubmitProof *prooftypes.MsgSubmitProof) error {
				sessionError := sessiontypes.ErrSessionInvalidSessionId.Wrapf(
					"%q",
					msgSubmitProof.GetSessionHeader().GetSessionId(),
				)
				return status.Error(
					codes.InvalidArgument,
					prooftypes.ErrProofInvalidSessionHeader.Wrapf("%s", sessionError).Error(),
				)
			},
		},
		{
			desc: "merkle proof cannot be empty",
			newProofMsg: func(t *testing.T) *prooftypes.MsgSubmitProof {
				// Construct new proof message.
				proof := newTestProofMsg(t,
					supplierOperatorAddr,
					validSessionHeader,
					validSessionTree,
					expectedMerkleProofPath,
				)

				// Set merkle proof to an empty byte slice.
				proof.Proof = []byte{}
				return proof
			},
			msgSubmitProofToExpectedErrorFn: func(_ *prooftypes.MsgSubmitProof) error {
				return status.Error(
					codes.InvalidArgument,
					prooftypes.ErrProofInvalidProof.Wrap(
						"proof cannot be empty",
					).Error(),
				)
			},
		},
		{
			desc: "proof session ID must match onchain session ID",
			newProofMsg: func(t *testing.T) *prooftypes.MsgSubmitProof {
				// Construct new proof message using the wrong session ID.
				return newTestProofMsg(t,
					supplierOperatorAddr,
					&wrongSessionIdHeader,
					validSessionTree,
					expectedMerkleProofPath,
				)
			},
			msgSubmitProofToExpectedErrorFn: func(msgSubmitProof *prooftypes.MsgSubmitProof) error {
				return status.Error(
					codes.FailedPrecondition,
					prooftypes.ErrProofInvalidSessionId.Wrapf(
						"session ID does not match onchain session ID; expected %q, got %q",
						validSessionHeader.GetSessionId(),
						msgSubmitProof.GetSessionHeader().GetSessionId(),
					).Error(),
				)
			},
		},
		{
			desc: "proof supplier must be in onchain session",
			newProofMsg: func(t *testing.T) *prooftypes.MsgSubmitProof {
				// Construct a proof message with a  supplier that does not belong in the session.
				return newTestProofMsg(t,
					wrongSupplierOperatorAddr,
					validSessionHeader,
					validSessionTree,
					expectedMerkleProofPath,
				)
			},
			msgSubmitProofToExpectedErrorFn: func(msgSubmitProof *prooftypes.MsgSubmitProof) error {
				return status.Error(
					codes.FailedPrecondition,
					prooftypes.ErrProofNotFound.Wrapf(
						"supplier operator address %q not found in session ID %q",
						wrongSupplierOperatorAddr,
						msgSubmitProof.GetSessionHeader().GetSessionId(),
					).Error(),
				)
			},
		},
	}

	// Submit the corresponding proof.
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			msgSubmitProof := test.newProofMsg(t)

			// Advance the block height to the proof path seed height.
			earliestSupplierProofCommitHeight := sharedtypes.GetEarliestSupplierProofCommitHeight(
				&sharedParams,
				msgSubmitProof.GetSessionHeader().GetSessionEndBlockHeight(),
				blockHeaderHash,
				msgSubmitProof.GetSupplierOperatorAddress(),
			)
			ctx = keepertest.SetBlockHeight(ctx, earliestSupplierProofCommitHeight-1)

			// Store proof path seed block hash in the session keeper so that it can
			// look it up during proof validation.
			keepers.StoreBlockHash(ctx)

			// Advance the block height to the earliest proof commit height.
			ctx = keepertest.SetBlockHeight(ctx, earliestSupplierProofCommitHeight)

			submitProofRes, err := srv.SubmitProof(ctx, msgSubmitProof)

			expectedErr := test.msgSubmitProofToExpectedErrorFn(msgSubmitProof)
			require.ErrorIs(t, err, expectedErr)
			require.ErrorContains(t, err, expectedErr.Error())
			require.Nil(t, submitProofRes)

			proofRes, err := keepers.AllProofs(ctx, &prooftypes.QueryAllProofsRequest{})
			require.NoError(t, err)

			// Expect zero proofs to have been persisted as all test cases are error cases.
			proofs := proofRes.GetProofs()
			require.Lenf(t, proofs, 0, "expected 0 proofs, got %d", len(proofs))

			// Assert that no proof submitted events were emitted.
			events := sdkCtx.EventManager().Events()
			proofSubmittedEvents := testutilevents.FilterEvents[*prooftypes.EventProofSubmitted](t, events)
			require.Equal(t, 0, len(proofSubmittedEvents))
		})
	}
}

func TestMsgServer_SubmitProof_FailSubmittingNonRequiredProof(t *testing.T) {
	opts := []keepertest.ProofKeepersOpt{
		// Set block hash so we can have a deterministic expected onchain proof requested by the protocol.
		keepertest.WithBlockHash(blockHeaderHash),
		// Set block height to 1 so there is a valid session onchain.
		keepertest.WithBlockHeight(1),
	}
	keepers, ctx := keepertest.NewProofModuleKeepers(t, opts...)
	sharedParams := keepers.SharedKeeper.GetParams(ctx)

	// Set proof keeper params to disable relay mining but never require a proof.
	proofParams := keepers.Keeper.GetParams(ctx)
	proofParams.ProofRequestProbability = 0
	err := keepers.Keeper.SetParams(ctx, proofParams)
	require.NoError(t, err)

	// Construct a keyring to hold the keypairs for the accounts used in the test.
	keyRing := keyring.NewInMemory(keepers.Codec)

	// Create a pre-generated account iterator to create accounts for the test.
	preGeneratedAccts := testkeyring.PreGeneratedAccounts()

	// Create accounts in the account keeper with corresponding keys in the
	// keyring for the application and supplier.
	supplierOperatorAddr := testkeyring.CreateOnChainAccount(
		ctx, t,
		supplierOperatorUid,
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

	fundSupplierOperatorAccount(t, ctx, keepers, supplierOperatorAddr)

	service := &sharedtypes.Service{
		Id:                   testServiceId,
		ComputeUnitsPerRelay: computeUnitsPerRelay,
		OwnerAddress:         sample.AccAddress(),
	}

	// Add a supplier and application pair that are expected to be in the session.
	keepers.AddServiceActors(ctx, t, service, supplierOperatorAddr, appAddr)

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
	numRelays := uint64(5)
	sessionTree := testtree.NewFilledSessionTree(
		ctx, t,
		numRelays, service.ComputeUnitsPerRelay,
		supplierOperatorUid, supplierOperatorAddr,
		sessionHeader, sessionHeader, sessionHeader,
		keyRing,
		ringClient,
	)

	// Advance the block height to the test claim msg height.
	claimMsgHeight := sharedtypes.GetEarliestSupplierClaimCommitHeight(
		&sharedParams,
		sessionHeader.GetSessionEndBlockHeight(),
		blockHeaderHash,
		supplierOperatorAddr,
	)
	ctx = keepertest.SetBlockHeight(ctx, claimMsgHeight)

	// Create a valid claim.
	createClaimAndStoreBlockHash(
		ctx, t, 1,
		supplierOperatorAddr,
		appAddr,
		service,
		sessionTree,
		sessionHeader,
		srv,
		keepers,
	)

	// Advance the block height to the proof path seed height.
	earliestSupplierProofCommitHeight := sharedtypes.GetEarliestSupplierProofCommitHeight(
		&sharedParams,
		sessionHeader.GetSessionEndBlockHeight(),
		blockHeaderHash,
		supplierOperatorAddr,
	)
	ctx = keepertest.SetBlockHeight(ctx, earliestSupplierProofCommitHeight-1)

	// Store proof path seed block hash in the session keeper so that it can
	// look it up during proof validation.
	keepers.StoreBlockHash(ctx)

	// Compute expected proof path.
	expectedMerkleProofPath = protocol.GetPathForProof(blockHeaderHash, sessionHeader.GetSessionId())

	// Advance the block height to the test proof msg height.
	proofMsgHeight := sharedtypes.GetEarliestSupplierProofCommitHeight(
		&sharedParams,
		sessionHeader.GetSessionEndBlockHeight(),
		blockHeaderHash,
		supplierOperatorAddr,
	)
	ctx = keepertest.SetBlockHeight(ctx, proofMsgHeight)

	proofMsg := newTestProofMsg(t,
		supplierOperatorAddr,
		sessionHeader,
		sessionTree,
		expectedMerkleProofPath,
	)
	submitProofRes, err := srv.SubmitProof(ctx, proofMsg)
	require.Nil(t, submitProofRes)
	require.ErrorIs(t, err, status.Error(codes.FailedPrecondition, prooftypes.ErrProofNotRequired.Error()))
}

// newTestProofMsg creates a new submit proof message that can be submitted
// to be validated and stored onchain.
func newTestProofMsg(
	t *testing.T,
	supplierOperatorAddr string,
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
		SupplierOperatorAddress: supplierOperatorAddr,
		SessionHeader:           sessionHeader,
		Proof:                   merkleProofBz,
	}
}

// createClaimAndStoreBlockHash creates a valid claim, submits it onchain,
// and on success, stores the block hash for retrieval at future heights.
// TODO_TECHDEBT(@bryanchriswhite): Consider if we could/should split
// this into two functions.
func createClaimAndStoreBlockHash(
	ctx context.Context,
	t *testing.T,
	sessionStartHeight int64,
	supplierOperatorAddr, appAddr string,
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
		supplierOperatorAddr,
		appAddr,
		service,
		merkleRootBz,
	)
	claimRes, err := msgServer.CreateClaim(ctx, claimMsg)
	require.NoError(t, err)

	sharedParams := keepers.SharedKeeper.GetParams(ctx)

	claimWindowOpenHeight := sharedtypes.GetClaimWindowOpenHeight(
		&sharedParams,
		sessionStartHeight,
	)

	ctx = keepertest.SetBlockHeight(ctx, claimWindowOpenHeight)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)

	earliestSupplierClaimCommitHeight := sharedtypes.GetEarliestSupplierClaimCommitHeight(
		&sharedParams,
		sessionStartHeight,
		sdkCtx.HeaderHash(),
		supplierOperatorAddr,
	)

	// Set block height to be after the session grace period.
	earliestSupplierClaimCommitCtx := keepertest.SetBlockHeight(ctx, earliestSupplierClaimCommitHeight)

	// Store the current context's block hash for future height, which is currently an EndBlocker operation.
	keepers.StoreBlockHash(earliestSupplierClaimCommitCtx)

	return claimRes.GetClaim()
}

// fundSupplierOperatorAccount sends enough coins to the supplier operator account
// to cover the cost of the proof submission.
func fundSupplierOperatorAccount(t *testing.T, ctx context.Context, keepers *keepertest.ProofModuleKeepers, supplierOperatorAddr string) {
	supplierOperatorAccAddr, err := sdk.AccAddressFromBech32(supplierOperatorAddr)
	require.NoError(t, err)

	err = keepers.SendCoinsFromModuleToAccount(
		ctx,
		suppliertypes.ModuleName,
		supplierOperatorAccAddr,
		types.NewCoins(types.NewCoin(volatile.DenomuPOKT, math.NewInt(100000000))),
	)
	require.NoError(t, err)

	coin := keepers.SpendableCoins(ctx, supplierOperatorAccAddr)
	require.Equal(t, coin[0].Amount, math.NewInt(100000000))

}
