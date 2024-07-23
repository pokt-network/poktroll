package keeper_test

import (
	"context"
	"encoding/hex"
	"os"
	"testing"

	"cosmossdk.io/depinject"
	ring_secp256k1 "github.com/athanorlabs/go-dleq/secp256k1"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/ring-go"
	"github.com/pokt-network/smt"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/crypto"
	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/pkg/crypto/rings"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/session"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/testkeyring"
	"github.com/pokt-network/poktroll/testutil/testrelayer"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	"github.com/pokt-network/poktroll/x/shared"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestIsProofValid_Error(t *testing.T) {
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

	// Get the session for the application/supplier pair which is
	// *not* expected to be claimed.
	unclaimedSessionHeader := keepers.GetSessionHeader(ctx, t, wrongAppAddr, wrongService, 1)

	// Construct a session header with session ID that doesn't match the expected session ID.
	wrongSessionIdHeader := *validSessionHeader
	wrongSessionIdHeader.SessionId = "wrong session ID"

	// TODO_TECHDEBT: add a test case such that we can distinguish between early
	// & late session end block heights.

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

	merkleRootBz, err := validSessionTree.Flush()
	require.NoError(t, err)

	claim := prooftypes.Claim{
		SessionHeader:   validSessionHeader,
		SupplierAddress: supplierAddr,
		RootHash:        merkleRootBz,
	}
	keepers.UpsertClaim(ctx, claim)

	// Compute the difficulty in bits of the closest relay from the valid session tree.
	validClosestRelayDifficultyBits := getClosestRelayDifficulty(t, validSessionTree, expectedMerkleProofPath)

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

	lowTargetHash, _ := hex.DecodeString("00000000000000000000000000000000000000000000000000000000000000ff")
	var lowTargetHashArr [protocol.RelayHasherSize]byte
	copy(lowTargetHashArr[:], lowTargetHash)
	highExpectedTargetDifficulty := protocol.GetDifficultyFromHash(lowTargetHashArr)

	tests := []struct {
		desc        string
		newProof    func(t *testing.T) *prooftypes.Proof
		expectedErr error
	}{
		{
			desc: "proof service ID cannot be empty",
			newProof: func(t *testing.T) *prooftypes.Proof {
				// Set proof session ID to empty string.
				emptySessionIdHeader := *validSessionHeader
				emptySessionIdHeader.SessionId = ""

				// Construct new proof message.
				return newProof(t,
					supplierAddr,
					&emptySessionIdHeader,
					validSessionTree,
					expectedMerkleProofPath)
			},
			expectedErr: prooftypes.ErrProofInvalidSessionId.Wrapf(
				"session ID does not match on-chain session ID; expected %q, got %q",
				validSessionHeader.GetSessionId(),
				"",
			),
		},
		{
			desc: "merkle proof cannot be empty",
			newProof: func(t *testing.T) *prooftypes.Proof {
				// Construct new proof message.
				proof := newProof(t,
					supplierAddr,
					validSessionHeader,
					validSessionTree,
					expectedMerkleProofPath,
				)

				// Set merkle proof to an empty byte slice.
				proof.ClosestMerkleProof = []byte{}
				return proof
			},
			expectedErr: prooftypes.ErrProofInvalidProof.Wrap(
				"proof cannot be empty",
			),
		},
		{
			desc: "proof session ID must match on-chain session ID",
			newProof: func(t *testing.T) *prooftypes.Proof {
				// Construct new proof message using the wrong session ID.
				return newProof(t,
					supplierAddr,
					&wrongSessionIdHeader,
					validSessionTree,
					expectedMerkleProofPath,
				)
			},
			expectedErr: prooftypes.ErrProofInvalidSessionId.Wrapf(
				"session ID does not match on-chain session ID; expected %q, got %q",
				validSessionHeader.GetSessionId(),
				wrongSessionIdHeader.GetSessionId(),
			),
		},
		{
			desc: "proof supplier must be in on-chain session",
			newProof: func(t *testing.T) *prooftypes.Proof {
				// Construct a proof message with a  supplier that does not belong in the session.
				return newProof(t,
					wrongSupplierAddr,
					validSessionHeader,
					validSessionTree,
					expectedMerkleProofPath,
				)
			},
			expectedErr: prooftypes.ErrProofNotFound.Wrapf(
				"supplier address %q not found in session ID %q",
				wrongSupplierAddr,
				validSessionHeader.GetSessionId(),
			),
		},
		{
			desc: "merkle proof must be deserializabled",
			newProof: func(t *testing.T) *prooftypes.Proof {
				// Construct new proof message.
				proof := newProof(t,
					supplierAddr,
					validSessionHeader,
					validSessionTree,
					expectedMerkleProofPath,
				)

				// Set merkle proof to an incorrect byte slice.
				proof.ClosestMerkleProof = invalidClosestProofBytes

				return proof
			},
			expectedErr: prooftypes.ErrProofInvalidProof.Wrapf(
				"failed to unmarshal closest merkle proof: %s",
				expectedInvalidProofUnmarshalErr,
			),
		},
		{
			desc: "relay must be deserializable",
			newProof: func(t *testing.T) *prooftypes.Proof {
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
				claim := newClaim(t,
					supplierAddr,
					validSessionHeader,
					mangledRelayMerkleRootBz,
				)
				keepers.UpsertClaim(claimCtx, *claim)
				require.NoError(t, err)

				// Construct new proof message derived from a session tree
				// with an unserializable relay.
				return newProof(t,
					supplierAddr,
					validSessionHeader,
					mangledRelaySessionTree,
					expectedMerkleProofPath,
				)
			},
			expectedErr: prooftypes.ErrProofInvalidRelay.Wrapf(
				"failed to unmarshal relay: %s",
				keepers.Codec.Unmarshal(mangledRelayBz, &servicetypes.Relay{}),
			),
		},
		{
			// TODO_TEST(community): expand: test case to cover each session header field.
			desc: "relay request session header must match proof session header",
			newProof: func(t *testing.T) *prooftypes.Proof {
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
				claim := newClaim(t,
					supplierAddr,
					validSessionHeader,
					wrongRequestSessionIdMerkleRootBz,
				)
				keepers.UpsertClaim(claimCtx, *claim)
				require.NoError(t, err)

				// Construct new proof message using the valid session header,
				// *not* the one used in the session tree's relay request.
				return newProof(t,
					supplierAddr,
					validSessionHeader,
					wrongRequestSessionIdSessionTree,
					expectedMerkleProofPath,
				)
			},
			expectedErr: prooftypes.ErrProofInvalidRelay.Wrapf(
				"session headers session IDs mismatch; expected: %q, got: %q",
				validSessionHeader.GetSessionId(),
				wrongSessionIdHeader.GetSessionId(),
			),
		},
		{
			// TODO_TEST: expand: test case to cover each session header field.
			desc: "relay response session header must match proof session header",
			newProof: func(t *testing.T) *prooftypes.Proof {
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
				claim := newClaim(t,
					supplierAddr,
					validSessionHeader,
					wrongResponseSessionIdMerkleRootBz,
				)
				keepers.UpsertClaim(claimCtx, *claim)
				require.NoError(t, err)

				// Construct new proof message using the valid session header,
				// *not* the one used in the session tree's relay response.
				return newProof(t,
					supplierAddr,
					validSessionHeader,
					wrongResponseSessionIdSessionTree,
					expectedMerkleProofPath,
				)
			},
			expectedErr: prooftypes.ErrProofInvalidRelay.Wrapf(
				"session headers session IDs mismatch; expected: %q, got: %q",
				validSessionHeader.GetSessionId(),
				wrongSessionIdHeader.GetSessionId(),
			),
		},
		{
			desc: "relay request signature must be valid",
			newProof: func(t *testing.T) *prooftypes.Proof {
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

				claim := newClaim(t,
					supplierAddr,
					validSessionHeader,
					invalidRequestSignatureMerkleRootBz,
				)
				keepers.UpsertClaim(claimCtx, *claim)
				require.NoError(t, err)

				// Construct new proof message derived from a session tree
				// with an invalid relay request signature.
				return newProof(t,
					supplierAddr,
					validSessionHeader,
					invalidRequestSignatureSessionTree,
					expectedMerkleProofPath,
				)
			},
			expectedErr: prooftypes.ErrProofInvalidRelayRequest.Wrapf(
				"error deserializing ring signature: %s",
				new(ring.RingSig).Deserialize(ring_secp256k1.NewCurve(), invalidSignatureBz),
			),
		},
		{
			desc: "relay request signature is valid but signed by an incorrect application",
			newProof: func(t *testing.T) *prooftypes.Proof {
				t.Skip("TODO_TECHDEBT(@bryanchriswhite): Implement this")
				return nil
			},
		},
		{
			desc: "relay response signature must be valid",
			newProof: func(t *testing.T) *prooftypes.Proof {
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
				claim := newClaim(t,
					supplierAddr,
					validSessionHeader,
					invalidResponseSignatureMerkleRootBz,
				)
				keepers.UpsertClaim(claimCtx, *claim)
				require.NoError(t, err)

				// Construct new proof message derived from a session tree
				// with an invalid relay response signature.
				return newProof(t,
					supplierAddr,
					validSessionHeader,
					invalidResponseSignatureSessionTree,
					expectedMerkleProofPath,
				)
			},
			expectedErr: servicetypes.ErrServiceInvalidRelayResponse.Wrap("invalid signature"),
		},
		{
			desc: "relay response signature is valid but signed by an incorrect supplier",
			newProof: func(t *testing.T) *prooftypes.Proof {
				t.Skip("TODO_TECHDEBT(@bryanchriswhite): Implement this")
				return nil
			},
		},
		{
			desc: "the merkle proof path provided does not match the one expected/enforced by the protocol",
			newProof: func(t *testing.T) *prooftypes.Proof {
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

				// Create an upsert the claim
				claim := newClaim(t,
					supplierAddr,
					validSessionHeader,
					wrongPathMerkleRootBz,
				)
				keepers.UpsertClaim(claimCtx, *claim)
				require.NoError(t, err)

				// Construct new proof message derived from a session tree
				// with an invalid relay response signature.
				return newProof(t, supplierAddr, validSessionHeader, wrongPathSessionTree, wrongClosestProofPath)
			},
			expectedErr: prooftypes.ErrProofInvalidProof.Wrapf(
				"the path of the proof provided (%x) does not match one expected by the on-chain protocol (%x)",
				wrongClosestProofPath,
				protocol.GetPathForProof(sdkCtx.HeaderHash(), validSessionHeader.GetSessionId()),
			),
		},
		{
			desc: "relay difficulty must be greater than or equal to minimum (zero difficulty)",
			newProof: func(t *testing.T) *prooftypes.Proof {
				// Set the minimum relay difficulty to a non-zero value such that the relays
				// constructed by the test helpers have a negligible chance of being valid.
				err = keepers.Keeper.SetParams(ctx, prooftypes.Params{
					RelayDifficultyTargetHash: lowTargetHash,
				})
				require.NoError(t, err)

				// Reset the minimum relay difficulty to zero after this test case.
				t.Cleanup(func() {
					err = keepers.Keeper.SetParams(ctx, prooftypes.DefaultParams())
					require.NoError(t, err)
				})

				// Construct a proof message with a session tree containing
				// a relay of insufficient difficulty.
				return newProof(t,
					supplierAddr,
					validSessionHeader,
					validSessionTree,
					expectedMerkleProofPath,
				)
			},
			expectedErr: prooftypes.ErrProofInvalidRelay.Wrapf(
				"the difficulty relay being proven is (%d), and is smaller than the target difficulty (%d) for service %s",
				validClosestRelayDifficultyBits,
				highExpectedTargetDifficulty,
				validSessionHeader.Service.Id,
			),
		},
		{
			desc: "relay difficulty must be greater than or equal to minimum (non-zero difficulty)",
			newProof: func(t *testing.T) *prooftypes.Proof {
				t.Skip("TODO_TECHDEBT(@bryanchriswhite): Implement this")
				return nil
			},
		},
		{
			desc: "claim must exist for proof message",
			newProof: func(t *testing.T) *prooftypes.Proof {
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
				return newProof(t,
					wrongSupplierAddr,
					unclaimedSessionHeader,
					unclaimedSessionTree,
					expectedMerkleProofPath,
				)
			},
			expectedErr: prooftypes.ErrProofClaimNotFound.Wrapf(
				"no claim found for session ID %q and supplier %q",
				unclaimedSessionHeader.GetSessionId(),
				wrongSupplierAddr,
			),
		},
		{
			desc: "Valid proof cannot validate claim with an incorrect root",
			newProof: func(t *testing.T) *prooftypes.Proof {
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
				claim := newClaim(t,
					supplierAddr,
					validSessionHeader,
					wrongMerkleRootBz,
				)
				keepers.UpsertClaim(claimCtx, *claim)
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

				return newProof(t,
					supplierAddr,
					validSessionHeader,
					validSessionTree,
					expectedMerkleProofPath,
				)
			},
			expectedErr: prooftypes.ErrProofInvalidProof.Wrap("invalid closest merkle proof"),
		},
		{
			desc: "claim and proof application addresses must match",
			newProof: func(t *testing.T) *prooftypes.Proof {
				t.Skip("this test case reduces to either the 'claim must exist for proof message' or 'proof session ID must match on-chain session ID cases")
				return nil
			},
		},
		{
			desc: "claim and proof service IDs must match",
			newProof: func(t *testing.T) *prooftypes.Proof {
				t.Skip("this test case reduces to either the 'claim must exist for proof message' or 'proof session ID must match on-chain session ID cases")
				return nil
			},
		},
		{
			desc: "claim and proof supplier addresses must match",
			newProof: func(t *testing.T) *prooftypes.Proof {
				t.Skip("this test case reduces to either the 'claim must exist for proof message' or 'proof session ID must match on-chain session ID cases")
				return nil
			},
		},
	}

	// Submit the corresponding proof.
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			proof := test.newProof(t)

			// Advance the block height to the proof path seed height.
			earliestSupplierProofCommitHeight := shared.GetEarliestSupplierProofCommitHeight(
				&sharedParams,
				proof.GetSessionHeader().GetSessionEndBlockHeight(),
				blockHeaderHash,
				proof.GetSupplierAddress(),
			)
			ctx = keepertest.SetBlockHeight(ctx, earliestSupplierProofCommitHeight-1)

			// Store proof path seed block hash in the session keeper so that it can
			// look it up during proof validation.
			keepers.StoreBlockHash(ctx)

			// Advance the block height to the earliest proof commit height.
			ctx = keepertest.SetBlockHeight(ctx, earliestSupplierProofCommitHeight)
			isProofValid, err := keepers.IsProofValid(ctx, proof)
			require.ErrorContains(t, err, test.expectedErr.Error())
			require.False(t, isProofValid)
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
	sessionTreeHeader, reqHeader, resHeader *sessiontypes.SessionHeader,
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
	sessionTreeHeader *sessiontypes.SessionHeader,
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
	sessionTree, err := session.NewSessionTree(
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
	reqHeader, resHeader *sessiontypes.SessionHeader,
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

// getClosestRelayDifficulty returns the mining difficulty number which corresponds
// to the relayHash stored in the sessionTree that is closest to the merkle proof
// path provided.
func getClosestRelayDifficulty(
	t *testing.T,
	sessionTree relayer.SessionTree,
	closestMerkleProofPath []byte,
) int64 {
	// Retrieve a merkle proof that is closest to the path provided
	closestMerkleProof, err := sessionTree.ProveClosest(closestMerkleProofPath)
	require.NoError(t, err)

	// Extract the Relay (containing the RelayResponse & RelayRequest) from the merkle proof.
	relay := new(servicetypes.Relay)
	relayBz := closestMerkleProof.GetValueHash(&protocol.SmtSpec)
	err = relay.Unmarshal(relayBz)
	require.NoError(t, err)

	// Retrieve the hash of the relay.
	relayHash, err := relay.GetHash()
	require.NoError(t, err)

	return protocol.GetDifficultyFromHash(relayHash)
}

// newProof creates a new proof structure.
func newProof(
	t *testing.T,
	supplierAddr string,
	sessionHeader *sessiontypes.SessionHeader,
	sessionTree relayer.SessionTree,
	closestProofPath []byte,
) *prooftypes.Proof {
	t.Helper()

	// Generate a closest proof from the session tree using closestProofPath.
	merkleProof, err := sessionTree.ProveClosest(closestProofPath)
	require.NoError(t, err)
	require.NotNil(t, merkleProof)

	// Serialize the closest merkle proof.
	merkleProofBz, err := merkleProof.Marshal()
	require.NoError(t, err)

	return &prooftypes.Proof{
		SupplierAddress:    supplierAddr,
		SessionHeader:      sessionHeader,
		ClosestMerkleProof: merkleProofBz,
	}
}

func newClaim(
	t *testing.T,
	supplierAddr string,
	sessionHeader *sessiontypes.SessionHeader,
	rootHash []byte,
) *prooftypes.Claim {
	// Create a new claim.
	return &prooftypes.Claim{
		SupplierAddress: supplierAddr,
		SessionHeader:   sessionHeader,
		RootHash:        rootHash,
	}
}
