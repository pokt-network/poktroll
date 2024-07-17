package keeper_test

import (
	"context"
	"os"
	"testing"

	"cosmossdk.io/depinject"
	ring_secp256k1 "github.com/athanorlabs/go-dleq/secp256k1"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/ring-go"
	"github.com/pokt-network/smt"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/pkg/crypto"
	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/pkg/crypto/rings"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/pkg/relayer"
	relayersession "github.com/pokt-network/poktroll/pkg/relayer/session"
	"github.com/pokt-network/poktroll/proto/types/proof"
	"github.com/pokt-network/poktroll/proto/types/service"
	"github.com/pokt-network/poktroll/proto/types/session"
	sharedtypes "github.com/pokt-network/poktroll/proto/types/shared"
	testutilevents "github.com/pokt-network/poktroll/testutil/events"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/testkeyring"
	"github.com/pokt-network/poktroll/testutil/testrelayer"
	"github.com/pokt-network/poktroll/x/proof/keeper"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	"github.com/pokt-network/poktroll/x/shared"
)

// TODO_TECHDEBT(@bryanchriswhite): Simplify this file; https://github.com/pokt-network/poktroll/pull/417#pullrequestreview-1958582600

const (
	supplierUid = "supplier"
)

var (
	blockHeaderHash         []byte
	expectedMerkleProofPath []byte

	// testProofParams sets:
	//  - the minimum relay difficulty bits to zero so that these tests don't need to mine for valid relays.
	//  - the proof request probability to 1 so that all test sessions require a proof.
	testProofParams = proof.Params{
		MinRelayDifficultyBits:  0,
		ProofRequestProbability: 1,
	}
)

func init() {
	// The CometBFT header hash is 32 bytes: https://docs.cometbft.com/main/spec/core/data_structures
	blockHeaderHash = make([]byte, 32)
	expectedMerkleProofPath = protocol.GetPathForProof(blockHeaderHash, "TODO_BLOCKER_session_id_currently_unused")
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

			// Set proof keeper params to disable relaymining and always require a proof.
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
			sessionTree := newFilledSessionTree(
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

			proofRes, err := keepers.AllProofs(ctx, &proof.QueryAllProofsRequest{})
			require.NoError(t, err)

			proofs := proofRes.GetProofs()
			require.Lenf(t, proofs, 1, "expected 1 proof, got %d", len(proofs))
			require.Equal(t, proofMsg.SessionHeader.SessionId, proofs[0].GetSessionHeader().GetSessionId())
			require.Equal(t, proofMsg.SupplierAddress, proofs[0].GetSupplierAddress())
			require.Equal(t, proofMsg.SessionHeader.GetSessionEndBlockHeight(), proofs[0].GetSessionHeader().GetSessionEndBlockHeight())

			events := sdkCtx.EventManager().Events()
			require.Equal(t, 2, len(events))

			proofSubmittedEvents := testutilevents.FilterEvents[*proof.EventProofSubmitted](t, events, "poktroll.proof.EventProofSubmitted")
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
	sessionTree := newFilledSessionTree(
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
				proof.ErrProofProofOutsideOfWindow.Wrapf(
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
				proof.ErrProofProofOutsideOfWindow.Wrapf(
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

			proofRes, err := keepers.AllProofs(ctx, &proof.QueryAllProofsRequest{})
			require.NoError(t, err)

			proofs := proofRes.GetProofs()
			require.Lenf(t, proofs, 0, "expected 0 proof, got %d", len(proofs))

			// Assert that only the create claim event was emitted.
			events := sdkCtx.EventManager().Events()
			require.Equal(t, 1, len(events))
			require.Equal(t, events[0].Type, "poktroll.proof.EventClaimCreated")
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

	// The base session start height used for testing
	sessionStartHeight := int64(1)

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

	testService := &sharedtypes.Service{Id: testServiceId}
	wrongService := &sharedtypes.Service{Id: "wrong_svc"}

	// Add a supplier and application pair that are expected to be in the session.
	keepers.AddServiceActors(ctx, t, testService, supplierAddr, appAddr)

	// Add a supplier and application pair that are *not* expected to be in the session.
	keepers.AddServiceActors(ctx, t, wrongService, wrongSupplierAddr, wrongAppAddr)

	// Get the session for the application/supplier pair which is expected
	// to be claimed and for which a valid proof would be accepted.
	validSessionHeader := keepers.GetSessionHeader(ctx, t, appAddr, testService, 1)

	// Get the session for the application/supplier pair which is
	// *not* expected to be claimed.
	unclaimedSessionHeader := keepers.GetSessionHeader(ctx, t, wrongAppAddr, wrongService, 1)

	// Construct a session header with session ID that doesn't match the expected session ID.
	wrongSessionIdHeader := *validSessionHeader
	wrongSessionIdHeader.SessionId = "wrong session ID"

	// TODO_TECHDEBT: add a test case such that we can distinguish between early
	// & late session end block heights.

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
	validSessionTree := newFilledSessionTree(
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
		testService,
		validSessionTree,
		validSessionHeader,
		srv,
		keepers,
	)

	// Compute the difficulty in bits of the closest relay from the valid session tree.
	validClosestRelayDifficultyBits := getClosestRelayDifficultyBits(t, validSessionTree, expectedMerkleProofPath)

	// Copy `emptyBlockHash` to `wrongClosestProofPath` to with a missing byte
	// so the closest proof is invalid (i.e. unmarshalable).
	invalidClosestProofBytes := make([]byte, len(expectedMerkleProofPath)-1)

	// Store the expected error returned during deserialization of the invalid
	// closest Merkle proof bytes.
	sparseMerkleClosestProof := &smt.SparseMerkleClosestProof{}
	expectedInvalidProofUnmarshalErr := sparseMerkleClosestProof.Unmarshal(invalidClosestProofBytes)

	// Construct a relay to be mangled such that it fails to deserialize in order
	// to set the error expectation for the relevant test case.
	mangledRelay := testrelayer.NewEmptyRelay(validSessionHeader, validSessionHeader, supplierAddr)

	// Ensure valid relay request and response signatures.
	testrelayer.SignRelayRequest(ctx, t, mangledRelay, appAddr, keyRing, ringClient)
	testrelayer.SignRelayResponse(ctx, t, mangledRelay, supplierUid, supplierAddr, keyRing)

	// Serialize the relay so that it can be mangled.
	mangledRelayBz, err := mangledRelay.Marshal()
	require.NoError(t, err)

	// Mangle the serialized relay to cause an error during deserialization.
	// Mangling could involve any byte randomly being swapped to any value
	// so unmarshaling fails, but we are setting the first byte to 0 for simplicity.
	mangledRelayBz[0] = 0x00

	// Declare an invalid signature byte slice to construct expected relay request
	// and response errors and use in corresponding test cases.
	invalidSignatureBz := []byte("invalid signature bytes")

	// Prepare an invalid proof of the correct size.
	wrongClosestProofPath := make([]byte, len(expectedMerkleProofPath))
	copy(wrongClosestProofPath, expectedMerkleProofPath)
	copy(wrongClosestProofPath, "wrong closest proof path")

	tests := []struct {
		desc        string
		newProofMsg func(t *testing.T) *proof.MsgSubmitProof
		expectedErr error
	}{
		{
			desc: "proof service ID cannot be empty",
			newProofMsg: func(t *testing.T) *proof.MsgSubmitProof {
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
				proof.ErrProofInvalidSessionId.Wrapf(
					"session ID does not match on-chain session ID; expected %q, got %q",
					validSessionHeader.GetSessionId(),
					"",
				).Error(),
			),
		},
		{
			desc: "merkle proof cannot be empty",
			newProofMsg: func(t *testing.T) *proof.MsgSubmitProof {
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
				proof.ErrProofInvalidProof.Wrap(
					"proof cannot be empty",
				).Error(),
			),
		},
		{
			desc: "proof session ID must match on-chain session ID",
			newProofMsg: func(t *testing.T) *proof.MsgSubmitProof {
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
				proof.ErrProofInvalidSessionId.Wrapf(
					"session ID does not match on-chain session ID; expected %q, got %q",
					validSessionHeader.GetSessionId(),
					wrongSessionIdHeader.GetSessionId(),
				).Error(),
			),
		},
		{
			desc: "proof supplier must be in on-chain session",
			newProofMsg: func(t *testing.T) *proof.MsgSubmitProof {
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
				proof.ErrProofNotFound.Wrapf(
					"supplier address %q not found in session ID %q",
					wrongSupplierAddr,
					validSessionHeader.GetSessionId(),
				).Error(),
			),
		},
		{
			desc: "merkle proof must be deserializable",
			newProofMsg: func(t *testing.T) *proof.MsgSubmitProof {
				// Construct new proof message.
				proof := newTestProofMsg(t,
					supplierAddr,
					validSessionHeader,
					validSessionTree,
					expectedMerkleProofPath,
				)

				// Set merkle proof to an incorrect byte slice.
				proof.Proof = invalidClosestProofBytes

				return proof
			},
			expectedErr: status.Error(
				codes.InvalidArgument,
				proof.ErrProofInvalidProof.Wrapf(
					"failed to unmarshal closest merkle proof: %s",
					expectedInvalidProofUnmarshalErr,
				).Error(),
			),
		},
		{
			desc: "relay must be deserializable",
			newProofMsg: func(t *testing.T) *proof.MsgSubmitProof {
				// Construct a session tree to which we'll add 1 unserializable relay.
				mangledRelaySessionTree := newEmptySessionTree(t, validSessionHeader, supplierAddr)

				// Add the mangled relay to the session tree.
				err = mangledRelaySessionTree.Update([]byte{1}, mangledRelayBz, 1)
				require.NoError(t, err)

				// Get the Merkle root for the session tree in order to construct a claim.
				mangledRelayMerkleRootBz, flushErr := mangledRelaySessionTree.Flush()
				require.NoError(t, flushErr)

				// Re-set the block height to the earliest claim commit height to create a new claim.
				claimCtx := cosmostypes.UnwrapSDKContext(ctx)
				claimCtx = claimCtx.WithBlockHeight(claimMsgHeight)

				// Create a claim with a merkle root derived from a session tree
				// with an unserializable relay.
				claimMsg := newTestClaimMsg(t,
					sessionStartHeight,
					validSessionHeader.GetSessionId(),
					supplierAddr,
					appAddr,
					testService,
					mangledRelayMerkleRootBz,
				)
				_, err = srv.CreateClaim(claimCtx, claimMsg)
				require.NoError(t, err)

				// Construct new proof message derived from a session tree
				// with an unserializable relay.
				return newTestProofMsg(t,
					supplierAddr,
					validSessionHeader,
					mangledRelaySessionTree,
					expectedMerkleProofPath,
				)
			},
			expectedErr: status.Error(
				codes.InvalidArgument,
				proof.ErrProofInvalidRelay.Wrapf(
					"failed to unmarshal relay: %s",
					keepers.Codec.Unmarshal(mangledRelayBz, &service.Relay{}),
				).Error(),
			),
		},
		{
			// TODO_TEST(community): expand: test case to cover each session header field.
			desc: "relay request session header must match proof session header",
			newProofMsg: func(t *testing.T) *proof.MsgSubmitProof {
				// Construct a session tree with 1 relay with a session header containing
				// a session ID that doesn't match the proof session ID.
				numRelays := uint(1)
				wrongRequestSessionIdSessionTree := newFilledSessionTree(
					ctx, t,
					numRelays,
					supplierUid, supplierAddr,
					validSessionHeader, &wrongSessionIdHeader, validSessionHeader,
					keyRing,
					ringClient,
				)

				// Get the Merkle root for the session tree in order to construct a claim.
				wrongRequestSessionIdMerkleRootBz, flushErr := wrongRequestSessionIdSessionTree.Flush()
				require.NoError(t, flushErr)

				// Re-set the block height to the earliest claim commit height to create a new claim.
				claimCtx := cosmostypes.UnwrapSDKContext(ctx)
				claimCtx = claimCtx.WithBlockHeight(claimMsgHeight)

				// Create a claim with a merkle root derived from a relay
				// request containing the wrong session ID.
				claimMsg := newTestClaimMsg(t,
					sessionStartHeight,
					validSessionHeader.GetSessionId(),
					supplierAddr,
					appAddr,
					testService,
					wrongRequestSessionIdMerkleRootBz,
				)
				_, err = srv.CreateClaim(claimCtx, claimMsg)
				require.NoError(t, err)

				// Construct new proof message using the valid session header,
				// *not* the one used in the session tree's relay request.
				return newTestProofMsg(t,
					supplierAddr,
					validSessionHeader,
					wrongRequestSessionIdSessionTree,
					expectedMerkleProofPath,
				)
			},
			expectedErr: status.Error(
				codes.FailedPrecondition,
				proof.ErrProofInvalidRelay.Wrapf(
					"session headers session IDs mismatch; expected: %q, got: %q",
					validSessionHeader.GetSessionId(),
					wrongSessionIdHeader.GetSessionId(),
				).Error(),
			),
		},
		{
			// TODO_TEST: expand: test case to cover each session header field.
			desc: "relay response session header must match proof session header",
			newProofMsg: func(t *testing.T) *proof.MsgSubmitProof {
				// Construct a session tree with 1 relay with a session header containing
				// a session ID that doesn't match the expected session ID.
				numRelays := uint(1)
				wrongResponseSessionIdSessionTree := newFilledSessionTree(
					ctx, t,
					numRelays,
					supplierUid, supplierAddr,
					validSessionHeader, validSessionHeader, &wrongSessionIdHeader,
					keyRing,
					ringClient,
				)

				// Get the Merkle root for the session tree in order to construct a claim.
				wrongResponseSessionIdMerkleRootBz, flushErr := wrongResponseSessionIdSessionTree.Flush()
				require.NoError(t, flushErr)

				// Re-set the block height to the earliest claim commit height to create a new claim.
				claimCtx := cosmostypes.UnwrapSDKContext(ctx)
				claimCtx = claimCtx.WithBlockHeight(claimMsgHeight)

				// Create a claim with a merkle root derived from a relay
				// response containing the wrong session ID.
				claimMsg := newTestClaimMsg(t,
					sessionStartHeight,
					validSessionHeader.GetSessionId(),
					supplierAddr,
					appAddr,
					testService,
					wrongResponseSessionIdMerkleRootBz,
				)
				_, err = srv.CreateClaim(claimCtx, claimMsg)
				require.NoError(t, err)

				// Construct new proof message using the valid session header,
				// *not* the one used in the session tree's relay response.
				return newTestProofMsg(t,
					supplierAddr,
					validSessionHeader,
					wrongResponseSessionIdSessionTree,
					expectedMerkleProofPath,
				)
			},
			expectedErr: status.Error(
				codes.FailedPrecondition,
				proof.ErrProofInvalidRelay.Wrapf(
					"session headers session IDs mismatch; expected: %q, got: %q",
					validSessionHeader.GetSessionId(),
					wrongSessionIdHeader.GetSessionId(),
				).Error(),
			),
		},
		{
			desc: "relay request signature must be valid",
			newProofMsg: func(t *testing.T) *proof.MsgSubmitProof {
				// Set the relay request signature to an invalid byte slice.
				invalidRequestSignatureRelay := testrelayer.NewEmptyRelay(validSessionHeader, validSessionHeader, supplierAddr)
				invalidRequestSignatureRelay.Req.Meta.Signature = invalidSignatureBz

				// Ensure a valid relay response signature.
				testrelayer.SignRelayResponse(ctx, t, invalidRequestSignatureRelay, supplierUid, supplierAddr, keyRing)

				invalidRequestSignatureRelayBz, marshalErr := invalidRequestSignatureRelay.Marshal()
				require.NoError(t, marshalErr)

				// Construct a session tree with 1 relay with a session header containing
				// a session ID that doesn't match the expected session ID.
				invalidRequestSignatureSessionTree := newEmptySessionTree(t, validSessionHeader, supplierAddr)

				// Add the relay to the session tree.
				err = invalidRequestSignatureSessionTree.Update([]byte{1}, invalidRequestSignatureRelayBz, 1)
				require.NoError(t, err)

				// Get the Merkle root for the session tree in order to construct a claim.
				invalidRequestSignatureMerkleRootBz, flushErr := invalidRequestSignatureSessionTree.Flush()
				require.NoError(t, flushErr)

				// Re-set the block height to the earliest claim commit height to create a new claim.
				claimCtx := cosmostypes.UnwrapSDKContext(ctx)
				claimCtx = claimCtx.WithBlockHeight(claimMsgHeight)

				// Create a claim with a merkle root derived from a session tree
				// with an invalid relay request signature.
				claimMsg := newTestClaimMsg(t,
					sessionStartHeight,
					validSessionHeader.GetSessionId(),
					supplierAddr,
					appAddr,
					testService,
					invalidRequestSignatureMerkleRootBz,
				)
				_, err = srv.CreateClaim(claimCtx, claimMsg)
				require.NoError(t, err)

				// Construct new proof message derived from a session tree
				// with an invalid relay request signature.
				return newTestProofMsg(t,
					supplierAddr,
					validSessionHeader,
					invalidRequestSignatureSessionTree,
					expectedMerkleProofPath,
				)
			},
			expectedErr: status.Error(
				codes.FailedPrecondition,
				proof.ErrProofInvalidRelayRequest.Wrapf(
					"error deserializing ring signature: %s",
					new(ring.RingSig).Deserialize(ring_secp256k1.NewCurve(), invalidSignatureBz),
				).Error(),
			),
		},
		{
			desc: "relay request signature is valid but signed by an incorrect application",
			newProofMsg: func(t *testing.T) *proof.MsgSubmitProof {
				t.Skip("TODO_TECHDEBT(@bryanchriswhite): Implement this")
				return nil
			},
		},
		{
			desc: "relay response signature must be valid",
			newProofMsg: func(t *testing.T) *proof.MsgSubmitProof {
				// Set the relay response signature to an invalid byte slice.
				relay := testrelayer.NewEmptyRelay(validSessionHeader, validSessionHeader, supplierAddr)
				relay.Res.Meta.SupplierSignature = invalidSignatureBz

				// Ensure a valid relay request signature
				testrelayer.SignRelayRequest(ctx, t, relay, appAddr, keyRing, ringClient)

				relayBz, marshalErr := relay.Marshal()
				require.NoError(t, marshalErr)

				// Construct a session tree with 1 relay with a session header containing
				// a session ID that doesn't match the expected session ID.
				invalidResponseSignatureSessionTree := newEmptySessionTree(t, validSessionHeader, supplierAddr)

				// Add the relay to the session tree.
				err = invalidResponseSignatureSessionTree.Update([]byte{1}, relayBz, 1)
				require.NoError(t, err)

				// Get the Merkle root for the session tree in order to construct a claim.
				invalidResponseSignatureMerkleRootBz, flushErr := invalidResponseSignatureSessionTree.Flush()
				require.NoError(t, flushErr)

				// Re-set the block height to the earliest claim commit height to create a new claim.
				claimCtx := cosmostypes.UnwrapSDKContext(ctx)
				claimCtx = claimCtx.WithBlockHeight(claimMsgHeight)

				// Create a claim with a merkle root derived from a session tree
				// with an invalid relay response signature.
				claimMsg := newTestClaimMsg(t,
					sessionStartHeight,
					validSessionHeader.GetSessionId(),
					supplierAddr,
					appAddr,
					testService,
					invalidResponseSignatureMerkleRootBz,
				)
				_, err = srv.CreateClaim(claimCtx, claimMsg)
				require.NoError(t, err)

				// Construct new proof message derived from a session tree
				// with an invalid relay response signature.
				return newTestProofMsg(t,
					supplierAddr,
					validSessionHeader,
					invalidResponseSignatureSessionTree,
					expectedMerkleProofPath,
				)
			},
			expectedErr: status.Error(
				codes.FailedPrecondition,
				service.ErrServiceInvalidRelayResponse.Wrap("invalid signature").Error(),
			),
		},
		{
			desc: "relay response signature is valid but signed by an incorrect supplier",
			newProofMsg: func(t *testing.T) *proof.MsgSubmitProof {
				t.Skip("TODO_TECHDEBT(@bryanchriswhite): Implement this")
				return nil
			},
		},
		{
			desc: "the merkle proof path provided does not match the one expected/enforced by the protocol",
			newProofMsg: func(t *testing.T) *proof.MsgSubmitProof {
				// Construct a new valid session tree for this test case because once the
				// closest proof has already been generated, the path cannot be changed.
				numRelays := uint(5)
				wrongPathSessionTree := newFilledSessionTree(
					ctx, t,
					numRelays,
					supplierUid, supplierAddr,
					validSessionHeader, validSessionHeader, validSessionHeader,
					keyRing,
					ringClient,
				)

				wrongPathMerkleRootBz, flushErr := wrongPathSessionTree.Flush()
				require.NoError(t, flushErr)

				// Re-set the block height to the earliest claim commit height to create a new claim.
				claimCtx := keepertest.SetBlockHeight(ctx, claimMsgHeight)

				// Create a valid claim with the expected merkle root.
				claimMsg := newTestClaimMsg(t,
					sessionStartHeight,
					validSessionHeader.GetSessionId(),
					supplierAddr,
					appAddr,
					testService,
					wrongPathMerkleRootBz,
				)
				_, err = srv.CreateClaim(claimCtx, claimMsg)
				require.NoError(t, err)

				// Construct new proof message derived from a session tree
				// with an invalid relay response signature.
				return newTestProofMsg(t, supplierAddr, validSessionHeader, wrongPathSessionTree, wrongClosestProofPath)
			},
			expectedErr: status.Error(
				codes.FailedPrecondition,
				proof.ErrProofInvalidProof.Wrapf(
					"the path of the proof provided (%x) does not match one expected by the on-chain protocol (%x)",
					wrongClosestProofPath,
					protocol.GetPathForProof(sdkCtx.HeaderHash(), validSessionHeader.GetSessionId()),
				).Error(),
			),
		},
		{
			desc: "relay difficulty must be greater than or equal to minimum (zero difficulty)",
			newProofMsg: func(t *testing.T) *proof.MsgSubmitProof {
				// Set the minimum relay difficulty to a non-zero value such that the relays
				// constructed by the test helpers have a negligable chance of being valid.
				err = keepers.Keeper.SetParams(ctx, proof.Params{
					MinRelayDifficultyBits: 10,
				})
				require.NoError(t, err)

				// Reset the minimum relay difficulty to zero after this test case.
				t.Cleanup(func() {
					err = keepers.Keeper.SetParams(ctx, proof.DefaultParams())
					require.NoError(t, err)
				})

				// Construct a proof message with a session tree containing
				// a relay of insufficient difficulty.
				return newTestProofMsg(t,
					supplierAddr,
					validSessionHeader,
					validSessionTree,
					expectedMerkleProofPath,
				)
			},
			expectedErr: status.Error(
				codes.FailedPrecondition,
				proof.ErrProofInvalidRelay.Wrapf(
					"relay difficulty %d is less than the minimum difficulty %d",
					validClosestRelayDifficultyBits,
					10,
				).Error(),
			),
		},
		{
			desc: "relay difficulty must be greater than or equal to minimum (non-zero difficulty)",
			newProofMsg: func(t *testing.T) *proof.MsgSubmitProof {
				t.Skip("TODO_TECHDEBT(@bryanchriswhite): Implement this")
				return nil
			},
		},
		{ // group: claim must exist for proof message
			desc: "claim must exist for proof message",
			newProofMsg: func(t *testing.T) *proof.MsgSubmitProof {
				// Construct a new session tree corresponding to the unclaimed session.
				numRelays := uint(5)
				unclaimedSessionTree := newFilledSessionTree(
					ctx, t,
					numRelays,
					"wrong_supplier", wrongSupplierAddr,
					unclaimedSessionHeader, unclaimedSessionHeader, unclaimedSessionHeader,
					keyRing,
					ringClient,
				)

				// Discard session tree Merkle root because no claim is being created.
				// Session tree must be closed (flushed) to compute closest Merkle Proof.
				_, err = unclaimedSessionTree.Flush()
				require.NoError(t, err)

				// Compute expected proof path for the unclaimed session.
				expectedMerkleProofPath := protocol.GetPathForProof(
					blockHeaderHash,
					unclaimedSessionHeader.GetSessionId(),
				)

				// Construct new proof message using the supplier & session header
				// from the session which is *not* expected to be claimed.
				return newTestProofMsg(t,
					wrongSupplierAddr,
					unclaimedSessionHeader,
					unclaimedSessionTree,
					expectedMerkleProofPath,
				)
			},
			expectedErr: status.Error(
				codes.FailedPrecondition,
				proof.ErrProofClaimNotFound.Wrapf(
					"no claim found for session ID %q and supplier %q",
					unclaimedSessionHeader.GetSessionId(),
					wrongSupplierAddr,
				).Error(),
			),
		},
		{
			desc: "Valid proof cannot validate claim with an incorrect root",
			newProofMsg: func(t *testing.T) *proof.MsgSubmitProof {
				numRelays := uint(10)
				wrongMerkleRootSessionTree := newFilledSessionTree(
					ctx, t,
					numRelays,
					supplierUid, supplierAddr,
					validSessionHeader, validSessionHeader, validSessionHeader,
					keyRing,
					ringClient,
				)

				wrongMerkleRootBz, err := wrongMerkleRootSessionTree.Flush()
				require.NoError(t, err)

				// Re-set the block height to the earliest claim commit height to create a new claim.
				claimCtx := keepertest.SetBlockHeight(ctx, claimMsgHeight)

				// Create a claim with the incorrect Merkle root.
				wrongMerkleRootClaimMsg := newTestClaimMsg(t,
					sessionStartHeight,
					validSessionHeader.GetSessionId(),
					supplierAddr,
					appAddr,
					testService,
					wrongMerkleRootBz,
				)
				_, err = srv.CreateClaim(claimCtx, wrongMerkleRootClaimMsg)
				require.NoError(t, err)

				// Construct a valid session tree with 5 relays.
				validSessionTree := newFilledSessionTree(
					ctx, t,
					uint(5),
					supplierUid, supplierAddr,
					validSessionHeader, validSessionHeader, validSessionHeader,
					keyRing,
					ringClient,
				)

				_, err = validSessionTree.Flush()
				require.NoError(t, err)

				// Compute expected proof path for the session.
				expectedMerkleProofPath := protocol.GetPathForProof(
					blockHeaderHash,
					validSessionHeader.GetSessionId(),
				)

				return newTestProofMsg(t,
					supplierAddr,
					validSessionHeader,
					validSessionTree,
					expectedMerkleProofPath,
				)
			},
			expectedErr: status.Error(
				codes.FailedPrecondition,
				proof.ErrProofInvalidProof.Wrap("invalid closest merkle proof").Error(),
			),
		},
		{
			desc: "claim and proof application addresses must match",
			newProofMsg: func(t *testing.T) *proof.MsgSubmitProof {
				t.Skip("this test case reduces to either the 'claim must exist for proof message' or 'proof session ID must match on-chain session ID cases")
				return nil
			},
		},
		{
			desc: "claim and proof service IDs must match",
			newProofMsg: func(t *testing.T) *proof.MsgSubmitProof {
				t.Skip("this test case reduces to either the 'claim must exist for proof message' or 'proof session ID must match on-chain session ID cases")
				return nil
			},
		},
		{
			desc: "claim and proof supplier addresses must match",
			newProofMsg: func(t *testing.T) *proof.MsgSubmitProof {
				t.Skip("this test case reduces to either the 'claim must exist for proof message' or 'proof session ID must match on-chain session ID cases")
				return nil
			},
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

			proofRes, err := keepers.AllProofs(ctx, &proof.QueryAllProofsRequest{})
			require.NoError(t, err)

			// Expect zero proofs to have been persisted as all test cases are error cases.
			proofs := proofRes.GetProofs()
			require.Lenf(t, proofs, 0, "expected 0 proofs, got %d", len(proofs))

			// Assert that no proof submitted events were emitted.
			events := sdkCtx.EventManager().Events()
			proofSubmittedEvents := testutilevents.FilterEvents[*proof.EventProofSubmitted](t, events, "poktroll.proof.EventProofSubmitted")
			require.Equal(t, 0, len(proofSubmittedEvents))
		})
	}
}

// newFilledSessionTree creates a new session tree with numRelays of relays
// filled out using the request and response headers provided where every
// relay is signed by the supplier and application respectively.
func newFilledSessionTree(
	ctx context.Context, t *testing.T,
	numRelays uint,
	supplierKeyUid, supplierAddr string,
	sessionTreeHeader, reqHeader, resHeader *session.SessionHeader,
	keyRing keyring.Keyring,
	ringClient crypto.RingClient,
) relayer.SessionTree {
	t.Helper()

	// Initialize an empty session tree with the given session header.
	sessionTree := newEmptySessionTree(t, sessionTreeHeader, supplierAddr)

	// Add numRelays of relays to the session tree.
	fillSessionTree(
		ctx, t,
		sessionTree, numRelays,
		supplierKeyUid, supplierAddr,
		reqHeader, resHeader,
		keyRing,
		ringClient,
	)

	return sessionTree
}

// newEmptySessionTree creates a new empty session tree with for given session.
func newEmptySessionTree(
	t *testing.T,
	sessionTreeHeader *session.SessionHeader,
	supplierAddr string,
) relayer.SessionTree {
	t.Helper()

	// Create a temporary session tree store directory for persistence.
	testSessionTreeStoreDir, err := os.MkdirTemp("", "session_tree_store_dir")
	require.NoError(t, err)

	// Delete the temporary session tree store directory after the test completes.
	t.Cleanup(func() {
		_ = os.RemoveAll(testSessionTreeStoreDir)
	})

	accAddress := cosmostypes.MustAccAddressFromBech32(supplierAddr)

	// Construct a session tree to add relays to and generate a proof from.
	sessionTree, err := relayersession.NewSessionTree(
		sessionTreeHeader,
		&accAddress,
		testSessionTreeStoreDir,
	)
	require.NoError(t, err)

	return sessionTree
}

// fillSessionTree fills the session tree with valid signed relays.
// A total of numRelays relays are added to the session tree with
// increasing weights (relay 1 has weight 1, relay 2 has weight 2, etc.).
func fillSessionTree(
	ctx context.Context, t *testing.T,
	sessionTree relayer.SessionTree,
	numRelays uint,
	supplierKeyUid, supplierAddr string,
	reqHeader, resHeader *session.SessionHeader,
	keyRing keyring.Keyring,
	ringClient crypto.RingClient,
) {
	t.Helper()

	for i := 0; i < int(numRelays); i++ {
		relay := testrelayer.NewSignedEmptyRelay(
			ctx, t,
			supplierKeyUid, supplierAddr,
			reqHeader, resHeader,
			keyRing,
			ringClient,
		)
		relayBz, err := relay.Marshal()
		require.NoError(t, err)

		relayKey, err := relay.GetHash()
		require.NoError(t, err)

		relayWeight := uint64(i)

		err = sessionTree.Update(relayKey[:], relayBz, relayWeight)
		require.NoError(t, err)
	}
}

// newTestProofMsg creates a new submit proof message that can be submitted
// to be validated and stored on-chain.
func newTestProofMsg(
	t *testing.T,
	supplierAddr string,
	sessionHeader *session.SessionHeader,
	sessionTree relayer.SessionTree,
	closestProofPath []byte,
) *proof.MsgSubmitProof {
	t.Helper()

	// Generate a closest proof from the session tree using closestProofPath.
	merkleProof, err := sessionTree.ProveClosest(closestProofPath)
	require.NoError(t, err)
	require.NotNil(t, merkleProof)

	// Serialize the closest merkle proof.
	merkleProofBz, err := merkleProof.Marshal()
	require.NoError(t, err)

	return &proof.MsgSubmitProof{
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
	sessionHeader *session.SessionHeader,
	msgServer proof.MsgServer,
	keepers *keepertest.ProofModuleKeepers,
) *proof.Claim {
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

// getClosestRelayDifficultyBits returns the number of leading 0s (i.e. relay
// mining difficulty bits) in the relayHash stored in the sessionTree that is
// is closest to the merkle proof path provided.
func getClosestRelayDifficultyBits(
	t *testing.T,
	sessionTree relayer.SessionTree,
	closestMerkleProofPath []byte,
) uint64 {
	// Retrieve a merkle proof that is closest to the path provided
	closestMerkleProof, err := sessionTree.ProveClosest(closestMerkleProofPath)
	require.NoError(t, err)

	// Extract the Relay (containing the RelayResponse & RelayRequest) from the merkle proof.
	relay := new(service.Relay)
	relayBz := closestMerkleProof.GetValueHash(&protocol.SmtSpec)
	err = relay.Unmarshal(relayBz)
	require.NoError(t, err)

	// Retrieve the hash of the relay.
	relayHash, err := relay.GetHash()
	require.NoError(t, err)

	// Count the number of leading 0s in the relay hash to determine its difficulty.
	return uint64(protocol.CountHashDifficultyBits(relayHash))
}
