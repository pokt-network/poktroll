package keeper_test

import (
	"context"
	"encoding/hex"
	"testing"

	"cosmossdk.io/depinject"
	"cosmossdk.io/log"
	ring_secp256k1 "github.com/athanorlabs/go-dleq/secp256k1"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/ring-go"
	"github.com/pokt-network/smt"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/pkg/crypto/rings"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testkeyring"
	"github.com/pokt-network/poktroll/testutil/testrelayer"
	"github.com/pokt-network/poktroll/testutil/testtree"
	"github.com/pokt-network/poktroll/x/proof/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicekeeper "github.com/pokt-network/poktroll/x/service/keeper"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	"github.com/pokt-network/poktroll/x/shared"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestEnsureValidProof_Error(t *testing.T) {
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
		ComputeUnitsPerRelay: 1,
		OwnerAddress:         sample.AccAddress(),
	}
	wrongService := &sharedtypes.Service{Id: "wrong_svc"}

	// Add a supplier and application pair that are expected to be in the session.
	keepers.AddServiceActors(ctx, t, service, supplierOperatorAddr, appAddr)

	// Add a supplier and application pair that are *not* expected to be in the session.
	keepers.AddServiceActors(ctx, t, wrongService, wrongSupplierOperatorAddr, wrongAppAddr)

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
	claimMsgHeight := shared.GetEarliestSupplierClaimCommitHeight(
		&sharedParams,
		validSessionHeader.GetSessionEndBlockHeight(),
		blockHeaderHash,
		supplierOperatorAddr,
	)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(claimMsgHeight)
	ctx = sdkCtx

	merkleRootBz, err := validSessionTree.Flush()
	require.NoError(t, err)

	claim := prooftypes.Claim{
		SessionHeader:           validSessionHeader,
		SupplierOperatorAddress: supplierOperatorAddr,
		RootHash:                merkleRootBz,
	}
	keepers.UpsertClaim(ctx, claim)

	// Copy `emptyBlockHash` to `wrongClosestProofPath` to with a missing byte
	// so the closest proof is invalid (i.e. unmarshalable).
	invalidClosestProofBytes := make([]byte, len(expectedMerkleProofPath)-1)

	// Store the expected error returned during deserialization of the invalid
	// closest Merkle proof bytes.
	sparseCompactMerkleClosestProof := &smt.SparseCompactMerkleClosestProof{}
	expectedInvalidProofUnmarshalErr := sparseCompactMerkleClosestProof.Unmarshal(invalidClosestProofBytes)

	// Construct a relay to be mangled such that it fails to deserialize in order
	// to set the error expectation for the relevant test case.
	mangledRelay := testrelayer.NewEmptyRelay(validSessionHeader, validSessionHeader, supplierOperatorAddr)

	// Ensure valid relay request and response signatures.
	testrelayer.SignRelayRequest(ctx, t, mangledRelay, appAddr, keyRing, ringClient)
	testrelayer.SignRelayResponse(ctx, t, mangledRelay, supplierOperatorUid, supplierOperatorAddr, keyRing)

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

	lowTargetHash, err := hex.DecodeString("00000000000000000000000000000000000000000000000000000000000000ff")
	require.NoError(t, err)

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
				return testtree.NewProof(t,
					supplierOperatorAddr,
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
				proof := testtree.NewProof(t,
					supplierOperatorAddr,
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
				return testtree.NewProof(t,
					supplierOperatorAddr,
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
				return testtree.NewProof(t,
					wrongSupplierOperatorAddr,
					validSessionHeader,
					validSessionTree,
					expectedMerkleProofPath,
				)
			},
			expectedErr: prooftypes.ErrProofNotFound.Wrapf(
				"supplier operator address %q not found in session ID %q",
				wrongSupplierOperatorAddr,
				validSessionHeader.GetSessionId(),
			),
		},
		{
			desc: "merkle proof must be deserializable",
			newProof: func(t *testing.T) *prooftypes.Proof {
				// Construct new proof message.
				proof := testtree.NewProof(t,
					supplierOperatorAddr,
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
				mangledRelaySessionTree := testtree.NewEmptySessionTree(t, validSessionHeader, supplierOperatorAddr)

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
				claim := testtree.NewClaim(t,
					supplierOperatorAddr,
					validSessionHeader,
					mangledRelayMerkleRootBz,
				)
				keepers.UpsertClaim(claimCtx, *claim)
				require.NoError(t, err)

				// Construct new proof message derived from a session tree
				// with an unserializable relay.
				return testtree.NewProof(t,
					supplierOperatorAddr,
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
				numRelays := uint64(1)
				wrongRequestSessionIdSessionTree := testtree.NewFilledSessionTree(
					ctx, t,
					numRelays, service.ComputeUnitsPerRelay,
					supplierOperatorUid, supplierOperatorAddr,
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
				claim := testtree.NewClaim(t,
					supplierOperatorAddr,
					validSessionHeader,
					wrongRequestSessionIdMerkleRootBz,
				)
				keepers.UpsertClaim(claimCtx, *claim)
				require.NoError(t, err)

				// Construct new proof message using the valid session header,
				// *not* the one used in the session tree's relay request.
				return testtree.NewProof(t,
					supplierOperatorAddr,
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
				numRelays := uint64(1)
				wrongResponseSessionIdSessionTree := testtree.NewFilledSessionTree(
					ctx, t,
					numRelays, service.ComputeUnitsPerRelay,
					supplierOperatorUid, supplierOperatorAddr,
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
				claim := testtree.NewClaim(t,
					supplierOperatorAddr,
					validSessionHeader,
					wrongResponseSessionIdMerkleRootBz,
				)
				keepers.UpsertClaim(claimCtx, *claim)
				require.NoError(t, err)

				// Construct new proof message using the valid session header,
				// *not* the one used in the session tree's relay response.
				return testtree.NewProof(t,
					supplierOperatorAddr,
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
				invalidRequestSignatureRelay := testrelayer.NewEmptyRelay(validSessionHeader, validSessionHeader, supplierOperatorAddr)
				invalidRequestSignatureRelay.Req.Meta.Signature = invalidSignatureBz

				// Ensure a valid relay response signature.
				testrelayer.SignRelayResponse(ctx, t, invalidRequestSignatureRelay, supplierOperatorUid, supplierOperatorAddr, keyRing)

				invalidRequestSignatureRelayBz, marshalErr := invalidRequestSignatureRelay.Marshal()
				require.NoError(t, marshalErr)

				// Construct a session tree with 1 relay with a session header containing
				// a session ID that doesn't match the expected session ID.
				invalidRequestSignatureSessionTree := testtree.NewEmptySessionTree(t, validSessionHeader, supplierOperatorAddr)

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

				claim := testtree.NewClaim(t,
					supplierOperatorAddr,
					validSessionHeader,
					invalidRequestSignatureMerkleRootBz,
				)
				keepers.UpsertClaim(claimCtx, *claim)
				require.NoError(t, err)

				// Construct new proof message derived from a session tree
				// with an invalid relay request signature.
				return testtree.NewProof(t,
					supplierOperatorAddr,
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
				relay := testrelayer.NewEmptyRelay(validSessionHeader, validSessionHeader, supplierOperatorAddr)
				relay.Res.Meta.SupplierOperatorSignature = invalidSignatureBz

				// Ensure a valid relay request signature
				testrelayer.SignRelayRequest(ctx, t, relay, appAddr, keyRing, ringClient)

				relayBz, marshalErr := relay.Marshal()
				require.NoError(t, marshalErr)

				// Construct a session tree with 1 relay with a session header containing
				// a session ID that doesn't match the expected session ID.
				invalidResponseSignatureSessionTree := testtree.NewEmptySessionTree(t, validSessionHeader, supplierOperatorAddr)

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
				claim := testtree.NewClaim(t,
					supplierOperatorAddr,
					validSessionHeader,
					invalidResponseSignatureMerkleRootBz,
				)
				keepers.UpsertClaim(claimCtx, *claim)
				require.NoError(t, err)

				// Construct new proof message derived from a session tree
				// with an invalid relay response signature.
				return testtree.NewProof(t,
					supplierOperatorAddr,
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
				numRelays := uint64(5)
				wrongPathSessionTree := testtree.NewFilledSessionTree(
					ctx, t,
					numRelays, service.ComputeUnitsPerRelay,
					supplierOperatorUid, supplierOperatorAddr,
					validSessionHeader, validSessionHeader, validSessionHeader,
					keyRing,
					ringClient,
				)

				wrongPathMerkleRootBz, flushErr := wrongPathSessionTree.Flush()
				require.NoError(t, flushErr)

				// Re-set the block height to the earliest claim commit height to create a new claim.
				claimCtx := keepertest.SetBlockHeight(ctx, claimMsgHeight)

				// Create an upsert the claim
				claim := testtree.NewClaim(t,
					supplierOperatorAddr,
					validSessionHeader,
					wrongPathMerkleRootBz,
				)
				keepers.UpsertClaim(claimCtx, *claim)
				require.NoError(t, err)

				// Construct new proof message derived from a session tree
				// with an invalid relay response signature.
				return testtree.NewProof(t, supplierOperatorAddr, validSessionHeader, wrongPathSessionTree, wrongClosestProofPath)
			},
			expectedErr: prooftypes.ErrProofInvalidProof.Wrapf(
				"the path of the proof provided (%x) does not match one expected by the on-chain protocol (%x)",
				wrongClosestProofPath,
				protocol.GetPathForProof(sdkCtx.HeaderHash(), validSessionHeader.GetSessionId()),
			),
		},
		{
			desc: "relay difficulty must be greater than or equal to a high difficulty (low target hash)",
			newProof: func(t *testing.T) *prooftypes.Proof {
				serviceId := validSessionHeader.GetServiceId()
				logger := log.NewNopLogger()
				setRelayMiningDifficultyHash(ctx, keepers.ServiceKeeper, serviceId, lowTargetHash, logger)
				// Reset the minimum relay difficulty to zero after this test case.
				t.Cleanup(func() {
					setRelayMiningDifficultyHash(ctx, keepers.ServiceKeeper, serviceId, protocol.BaseRelayDifficultyHashBz, logger)
				})

				// Construct a proof message with a session tree containing
				// a valid relay but of insufficient difficulty.
				proof := testtree.NewProof(t,
					supplierOperatorAddr,
					validSessionHeader,
					validSessionTree,
					expectedMerkleProofPath,
				)

				// Extract relayHash to check below that it's difficulty is insufficient
				err = sparseCompactMerkleClosestProof.Unmarshal(proof.ClosestMerkleProof)
				require.NoError(t, err)
				var sparseMerkleClosestProof *smt.SparseMerkleClosestProof
				sparseMerkleClosestProof, err = smt.DecompactClosestProof(sparseCompactMerkleClosestProof, &protocol.SmtSpec)
				require.NoError(t, err)

				relayBz := sparseMerkleClosestProof.GetValueHash(&protocol.SmtSpec)
				relayHashArr := protocol.GetRelayHashFromBytes(relayBz)
				relayHash := relayHashArr[:]

				// Check that the relay difficulty is insufficient
				// DEV_NOTE: We are doing this validation in the "newProof" function
				// because of the scoping complexities of including it in expectedErr.
				isRelayVolumeApplicable := protocol.IsRelayVolumeApplicable(relayHash, lowTargetHash)
				require.False(t, isRelayVolumeApplicable)

				return proof

			},
			expectedErr: types.ErrProofInvalidRelayDifficulty, // Asserting on the default error but validation of values is done above
		},
		{
			desc: "claim must exist for proof message",
			newProof: func(t *testing.T) *prooftypes.Proof {
				// Construct a new session tree corresponding to the unclaimed session.
				numRelays := uint64(5)
				unclaimedSessionTree := testtree.NewFilledSessionTree(
					ctx, t,
					numRelays, service.ComputeUnitsPerRelay,
					"wrong_supplier", wrongSupplierOperatorAddr,
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
				return testtree.NewProof(t,
					wrongSupplierOperatorAddr,
					unclaimedSessionHeader,
					unclaimedSessionTree,
					expectedMerkleProofPath,
				)
			},
			expectedErr: prooftypes.ErrProofClaimNotFound.Wrapf(
				"no claim found for session ID %q and supplier %q",
				unclaimedSessionHeader.GetSessionId(),
				wrongSupplierOperatorAddr,
			),
		},
		{
			desc: "Valid proof cannot validate claim with an incorrect root",
			newProof: func(t *testing.T) *prooftypes.Proof {
				numRelays := uint64(10)
				wrongMerkleRootSessionTree := testtree.NewFilledSessionTree(
					ctx, t,
					numRelays, service.ComputeUnitsPerRelay,
					supplierOperatorUid, supplierOperatorAddr,
					validSessionHeader, validSessionHeader, validSessionHeader,
					keyRing,
					ringClient,
				)

				wrongMerkleRootBz, err := wrongMerkleRootSessionTree.Flush()
				require.NoError(t, err)

				// Re-set the block height to the earliest claim commit height to create a new claim.
				claimCtx := keepertest.SetBlockHeight(ctx, claimMsgHeight)

				// Create a claim with the incorrect Merkle root.
				claim := testtree.NewClaim(t,
					supplierOperatorAddr,
					validSessionHeader,
					wrongMerkleRootBz,
				)
				keepers.UpsertClaim(claimCtx, *claim)
				require.NoError(t, err)

				// Construct a valid session tree.
				numRelays = uint64(5)
				validSessionTree := testtree.NewFilledSessionTree(
					ctx, t,
					numRelays, service.ComputeUnitsPerRelay,
					supplierOperatorUid, supplierOperatorAddr,
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

				return testtree.NewProof(t,
					supplierOperatorAddr,
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
			desc: "claim and proof supplier operator addresses must match",
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
				proof.GetSupplierOperatorAddress(),
			)
			ctx = keepertest.SetBlockHeight(ctx, earliestSupplierProofCommitHeight-1)

			// Store proof path seed block hash in the session keeper so that it can
			// look it up during proof validation.
			keepers.StoreBlockHash(ctx)

			// Advance the block height to the earliest proof commit height.
			ctx = keepertest.SetBlockHeight(ctx, earliestSupplierProofCommitHeight)
			err := keepers.EnsureValidProof(ctx, proof)
			require.ErrorContains(t, err, test.expectedErr.Error())
		})
	}
}

func setRelayMiningDifficultyHash(
	ctx context.Context,
	serviceKeeper prooftypes.ServiceKeeper,
	serviceId string,
	targetHash []byte,
	logger log.Logger,
) {
	relayMiningDifficulty := servicekeeper.NewDefaultRelayMiningDifficulty(ctx, logger, serviceId, 0)
	relayMiningDifficulty.TargetHash = targetHash
	serviceKeeper.SetRelayMiningDifficulty(ctx, relayMiningDifficulty)
}
