package keeper_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"cosmossdk.io/depinject"
	ring_secp256k1 "github.com/athanorlabs/go-dleq/secp256k1"
	ringtypes "github.com/athanorlabs/go-dleq/types"
	cosmoscrypto "github.com/cosmos/cosmos-sdk/crypto"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/noot/ring-go"
	"github.com/pokt-network/smt"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/pkg/crypto"
	"github.com/pokt-network/poktroll/pkg/crypto/rings"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/protocol"
	"github.com/pokt-network/poktroll/pkg/relayer/session"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/x/proof/keeper"
	"github.com/pokt-network/poktroll/x/proof/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// TODO_TECHDEBT(@bryanchriswhite): Simplify this file; https://github.com/pokt-network/poktroll/pull/417#pullrequestreview-1958582600

const (
	supplierUid = "supplier"
)

var (
	blockHeaderHash         []byte
	expectedMerkleProofPath []byte
)

func init() {
	// The CometBFT header hash is 32 bytes: https://docs.cometbft.com/main/spec/core/data_structures
	blockHeaderHash = make([]byte, 32)
	expectedMerkleProofPath = keeper.GetPathForProof(blockHeaderHash, "TODO_BLOCKER_session_id_currently_unused")
}

func TestMsgServer_SubmitProof_Success(t *testing.T) {
	opts := []keepertest.ProofKeepersOpt{
		// Set block hash so we can have a deterministic expected on-chain proof requested by the protocol.
		keepertest.WithBlockHash(blockHeaderHash),
		// Set block height to 1 so there is a valid session on-chain.
		keepertest.WithBlockHeight(1),
	}
	keepers, ctx := keepertest.NewProofModuleKeepers(t, opts...)

	// Ensure the minimum relay difficulty bits is set to zero so this test
	// doesn't need to mine for valid relays.
	err := keepers.Keeper.SetParams(ctx, types.NewParams(0))
	require.NoError(t, err)

	// Construct a keyring to hold the keypairs for the accounts used in the test.
	keyRing := keyring.NewInMemory(keepers.Codec)

	// Create accounts in the account keeper with corresponding keys in the keyring for the application and supplier.
	supplierAddr := createAccount(ctx, t, supplierUid, keyRing, keepers).GetAddress().String()
	appAddr := createAccount(ctx, t, "app", keyRing, keepers).GetAddress().String()

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
		types.NewAppKeeperQueryClient(keepers.ApplicationKeeper),
		types.NewAccountKeeperQueryClient(keepers.AccountKeeper),
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

	// Create a valid claim.
	createClaimAndStoreBlockHash(
		ctx, t,
		supplierAddr,
		appAddr,
		service,
		sessionTree,
		sessionHeader,
		srv,
		keepers,
	)

	proofMsg := newTestProofMsg(t,
		supplierAddr,
		sessionHeader,
		sessionTree,
		expectedMerkleProofPath,
	)
	submitProofRes, err := srv.SubmitProof(ctx, proofMsg)
	require.NoError(t, err)
	require.NotNil(t, submitProofRes)

	proofRes, err := keepers.AllProofs(ctx, &types.QueryAllProofsRequest{})
	require.NoError(t, err)

	proofs := proofRes.GetProofs()
	require.Lenf(t, proofs, 1, "expected 1 proof, got %d", len(proofs))
	require.Equal(t, proofMsg.SessionHeader.SessionId, proofs[0].GetSessionHeader().GetSessionId())
	require.Equal(t, proofMsg.SupplierAddress, proofs[0].GetSupplierAddress())
	require.Equal(t, proofMsg.SessionHeader.GetSessionEndBlockHeight(), proofs[0].GetSessionHeader().GetSessionEndBlockHeight())
}

func TestMsgServer_SubmitProof_Error(t *testing.T) {
	opts := []keepertest.ProofKeepersOpt{
		// Set block hash such that on-chain closest merkle proof validation
		// uses the expected path.
		keepertest.WithBlockHash(expectedMerkleProofPath),
		// Set block height to 1 so there is a valid session on-chain.
		keepertest.WithBlockHeight(1),
	}
	keepers, ctx := keepertest.NewProofModuleKeepers(t, opts...)

	// Ensure the minimum relay difficulty bits is set to zero so that test cases
	// don't need to mine for valid relays.
	err := keepers.Keeper.SetParams(ctx, types.NewParams(0))
	require.NoError(t, err)

	// Construct a keyring to hold the keypairs for the accounts used in the test.
	keyRing := keyring.NewInMemory(keepers.Codec)

	// Create accounts in the account keeper with corresponding keys in the keyring
	// for the applications and suppliers used in the tests.
	supplierAddr := createAccount(ctx, t, supplierUid, keyRing, keepers).GetAddress().String()
	wrongSupplierAddr := createAccount(ctx, t, "wrong_supplier", keyRing, keepers).GetAddress().String()
	appAddr := createAccount(ctx, t, "app", keyRing, keepers).GetAddress().String()
	wrongAppAddr := createAccount(ctx, t, "wrong_app", keyRing, keepers).GetAddress().String()

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

	// Construct a session header with a session start height that doesn't match
	// the expected session start height.
	wrongSessionStartHeightHeader := *validSessionHeader
	wrongSessionStartHeightHeader.SessionStartBlockHeight = 2

	// Construct a session header with a session end height that doesn't match
	// the expected session end height.
	wrongSessionEndHeightHeader := *validSessionHeader
	wrongSessionEndHeightHeader.SessionEndBlockHeight = 3

	// TODO_TECHDEBT: add a test case such that we can distinguish between early
	// & late session end block heights.

	// Construct a proof message server from the proof keeper.
	srv := keeper.NewMsgServerImpl(*keepers.Keeper)

	// Construct a ringClient to get the application's ring & verify the relay
	// request signature.
	ringClient, err := rings.NewRingClient(depinject.Supply(
		polyzero.NewLogger(),
		types.NewAppKeeperQueryClient(keepers.ApplicationKeeper),
		types.NewAccountKeeperQueryClient(keepers.AccountKeeper),
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

	// Create a valid claim for the expected session and update the block hash
	// store for the corresponding session.
	createClaimAndStoreBlockHash(
		ctx, t,
		supplierAddr,
		appAddr,
		service,
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
	mangledRelay := newEmptyRelay(validSessionHeader, validSessionHeader)

	// Ensure valid relay request and response signatures.
	signRelayRequest(ctx, t, mangledRelay, appAddr, keyRing, ringClient)
	signRelayResponse(ctx, t, mangledRelay, supplierUid, supplierAddr, keyRing)

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
		newProofMsg func(t *testing.T) *types.MsgSubmitProof
		expectedErr error
	}{
		{
			desc: "proof service ID cannot be empty",
			newProofMsg: func(t *testing.T) *types.MsgSubmitProof {
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
				types.ErrProofInvalidSessionId.Wrapf(
					"session ID does not match on-chain session ID; expected %q, got %q",
					validSessionHeader.GetSessionId(),
					"",
				).Error(),
			),
		},
		{
			desc: "merkle proof cannot be empty",
			newProofMsg: func(t *testing.T) *types.MsgSubmitProof {
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
				types.ErrProofInvalidProof.Wrap(
					"proof cannot be empty",
				).Error(),
			),
		},
		{
			desc: "proof session ID must match on-chain session ID",
			newProofMsg: func(t *testing.T) *types.MsgSubmitProof {
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
				types.ErrProofInvalidSessionId.Wrapf(
					"session ID does not match on-chain session ID; expected %q, got %q",
					validSessionHeader.GetSessionId(),
					wrongSessionIdHeader.GetSessionId(),
				).Error(),
			),
		},
		{
			desc: "proof supplier must be in on-chain session",
			newProofMsg: func(t *testing.T) *types.MsgSubmitProof {
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
				types.ErrProofNotFound.Wrapf(
					"supplier address %q not found in session ID %q",
					wrongSupplierAddr,
					validSessionHeader.GetSessionId(),
				).Error(),
			),
		},
		{
			desc: "merkle proof must be deserializable",
			newProofMsg: func(t *testing.T) *types.MsgSubmitProof {
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
				types.ErrProofInvalidProof.Wrapf(
					"failed to unmarshal closest merkle proof: %s",
					expectedInvalidProofUnmarshalErr,
				).Error(),
			),
		},
		{
			desc: "relay must be deserializable",
			newProofMsg: func(t *testing.T) *types.MsgSubmitProof {
				// Construct a session tree to which we'll add 1 unserializable relay.
				mangledRelaySessionTree := newEmptySessionTree(t, validSessionHeader)

				// Add the mangled relay to the session tree.
				err = mangledRelaySessionTree.Update([]byte{1}, mangledRelayBz, 1)
				require.NoError(t, err)

				// Get the Merkle root for the session tree in order to construct a claim.
				mangledRelayMerkleRootBz, err := mangledRelaySessionTree.Flush()
				require.NoError(t, err)

				// Create a claim with a merkle root derived from a session tree
				// with an unserializable relay.
				claimMsg := newTestClaimMsg(t,
					validSessionHeader.GetSessionId(),
					supplierAddr,
					appAddr,
					service,
					mangledRelayMerkleRootBz,
				)
				_, err = srv.CreateClaim(ctx, claimMsg)
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
				types.ErrProofInvalidRelay.Wrapf(
					"failed to unmarshal relay: %s",
					keepers.Codec.Unmarshal(mangledRelayBz, &servicetypes.Relay{}),
				).Error(),
			),
		},
		{
			// TODO_TEST(community): expand: test case to cover each session header field.
			desc: "relay request session header must match proof session header",
			newProofMsg: func(t *testing.T) *types.MsgSubmitProof {
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
				wrongRequestSessionIdMerkleRootBz, err := wrongRequestSessionIdSessionTree.Flush()
				require.NoError(t, err)

				// Create a claim with a merkle root derived from a relay
				// request containing the wrong session ID.
				claimMsg := newTestClaimMsg(t,
					validSessionHeader.GetSessionId(),
					supplierAddr,
					appAddr,
					service,
					wrongRequestSessionIdMerkleRootBz,
				)
				_, err = srv.CreateClaim(ctx, claimMsg)
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
				types.ErrProofInvalidRelay.Wrapf(
					"session headers session IDs mismatch; expected: %q, got: %q",
					validSessionHeader.GetSessionId(),
					wrongSessionIdHeader.GetSessionId(),
				).Error(),
			),
		},
		{
			// TODO_TEST(community): expand: test case to cover each session header field.
			desc: "relay response session header must match proof session header",
			newProofMsg: func(t *testing.T) *types.MsgSubmitProof {
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
				wrongResponseSessionIdMerkleRootBz, err := wrongResponseSessionIdSessionTree.Flush()
				require.NoError(t, err)

				// Create a claim with a merkle root derived from a relay
				// response containing the wrong session ID.
				claimMsg := newTestClaimMsg(t,
					validSessionHeader.GetSessionId(),
					supplierAddr,
					appAddr,
					service,
					wrongResponseSessionIdMerkleRootBz,
				)
				_, err = srv.CreateClaim(ctx, claimMsg)
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
				types.ErrProofInvalidRelay.Wrapf(
					"session headers session IDs mismatch; expected: %q, got: %q",
					validSessionHeader.GetSessionId(),
					wrongSessionIdHeader.GetSessionId(),
				).Error(),
			),
		},
		{
			desc: "relay request signature must be valid",
			newProofMsg: func(t *testing.T) *types.MsgSubmitProof {
				// Set the relay request signature to an invalid byte slice.
				invalidRequestSignatureRelay := newEmptyRelay(validSessionHeader, validSessionHeader)
				invalidRequestSignatureRelay.Req.Meta.Signature = invalidSignatureBz

				// Ensure a valid relay response signature.
				signRelayResponse(ctx, t, invalidRequestSignatureRelay, supplierUid, supplierAddr, keyRing)

				invalidRequestSignatureRelayBz, err := invalidRequestSignatureRelay.Marshal()
				require.NoError(t, err)

				// Construct a session tree with 1 relay with a session header containing
				// a session ID that doesn't match the expected session ID.
				invalidRequestSignatureSessionTree := newEmptySessionTree(t, validSessionHeader)

				// Add the relay to the session tree.
				err = invalidRequestSignatureSessionTree.Update([]byte{1}, invalidRequestSignatureRelayBz, 1)
				require.NoError(t, err)

				// Get the Merkle root for the session tree in order to construct a claim.
				invalidRequestSignatureMerkleRootBz, err := invalidRequestSignatureSessionTree.Flush()
				require.NoError(t, err)

				// Create a claim with a merkle root derived from a session tree
				// with an invalid relay request signature.
				claimMsg := newTestClaimMsg(t,
					validSessionHeader.GetSessionId(),
					supplierAddr,
					appAddr,
					service,
					invalidRequestSignatureMerkleRootBz,
				)
				_, err = srv.CreateClaim(ctx, claimMsg)
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
				types.ErrProofInvalidRelayRequest.Wrapf(
					"error deserializing ring signature: %s",
					new(ring.RingSig).Deserialize(ring_secp256k1.NewCurve(), invalidSignatureBz),
				).Error(),
			),
		},
		{
			desc: "relay request signature is valid but signed by an incorrect application",
			newProofMsg: func(t *testing.T) *types.MsgSubmitProof {
				t.Skip("TODO_TECHDEBT(@bryanchriswhite): Implement this")
				return nil
			},
		},
		{
			desc: "relay response signature must be valid",
			newProofMsg: func(t *testing.T) *types.MsgSubmitProof {
				// Set the relay response signature to an invalid byte slice.
				relay := newEmptyRelay(validSessionHeader, validSessionHeader)
				relay.Res.Meta.SupplierSignature = invalidSignatureBz

				// Ensure a valid relay request signature
				signRelayRequest(ctx, t, relay, appAddr, keyRing, ringClient)

				relayBz, err := relay.Marshal()
				require.NoError(t, err)

				// Construct a session tree with 1 relay with a session header containing
				// a session ID that doesn't match the expected session ID.
				invalidResponseSignatureSessionTree := newEmptySessionTree(t, validSessionHeader)

				// Add the relay to the session tree.
				err = invalidResponseSignatureSessionTree.Update([]byte{1}, relayBz, 1)
				require.NoError(t, err)

				// Get the Merkle root for the session tree in order to construct a claim.
				invalidResponseSignatureMerkleRootBz, err := invalidResponseSignatureSessionTree.Flush()
				require.NoError(t, err)

				// Create a claim with a merkle root derived from a session tree
				// with an invalid relay response signature.
				claimMsg := newTestClaimMsg(t,
					validSessionHeader.GetSessionId(),
					supplierAddr,
					appAddr,
					service,
					invalidResponseSignatureMerkleRootBz,
				)
				_, err = srv.CreateClaim(ctx, claimMsg)
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
				servicetypes.ErrServiceInvalidRelayResponse.Wrap("invalid signature").Error(),
			),
		},
		{
			desc: "relay response signature is valid but signed by an incorrect supplier",
			newProofMsg: func(t *testing.T) *types.MsgSubmitProof {
				t.Skip("TODO_TECHDEBT(@bryanchriswhite): Implement this")
				return nil
			},
		},
		{
			desc: "the merkle proof path provided does not match the one expected/enforced by the protocol",
			newProofMsg: func(t *testing.T) *types.MsgSubmitProof {
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

				wrongPathMerkleRootBz, err := wrongPathSessionTree.Flush()
				require.NoError(t, err)

				// Create a valid claim with the expected merkle root.
				claimMsg := newTestClaimMsg(t,
					validSessionHeader.GetSessionId(),
					supplierAddr,
					appAddr,
					service,
					wrongPathMerkleRootBz,
				)
				_, err = srv.CreateClaim(ctx, claimMsg)
				require.NoError(t, err)

				// Construct new proof message derived from a session tree
				// with an invalid relay response signature.
				return newTestProofMsg(t, supplierAddr, validSessionHeader, wrongPathSessionTree, wrongClosestProofPath)
			},
			expectedErr: status.Error(
				codes.FailedPrecondition,
				types.ErrProofInvalidProof.Wrapf(
					"the proof for the path provided (%x) does not match one expected by the on-chain protocol (%x)",
					wrongClosestProofPath,
					blockHeaderHash,
				).Error(),
			),
		},
		{
			desc: "relay difficulty must be greater than or equal to minimum (zero difficulty)",
			newProofMsg: func(t *testing.T) *types.MsgSubmitProof {
				// Set the minimum relay difficulty to a non-zero value such that the relays
				// constructed by the test helpers have a negligable chance of being valid.
				err := keepers.Keeper.SetParams(ctx, types.Params{
					MinRelayDifficultyBits: 10,
				})
				require.NoError(t, err)

				// Reset the minimum relay difficulty to zero after this test case.
				t.Cleanup(func() {
					err := keepers.Keeper.SetParams(ctx, types.DefaultParams())
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
				types.ErrProofInvalidRelay.Wrapf(
					"relay difficulty %d is less than the minimum difficulty %d",
					validClosestRelayDifficultyBits,
					10,
				).Error(),
			),
		},
		{
			desc: "relay difficulty must be greater than or equal to minimum (non-zero difficulty)",
			newProofMsg: func(t *testing.T) *types.MsgSubmitProof {
				t.Skip("TODO_TECHDEBT(@bryanchriswhite): Implement this")
				return nil
			},
		},
		{ // group: claim must exist for proof message
			desc: "claim must exist for proof message",
			newProofMsg: func(t *testing.T) *types.MsgSubmitProof {
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
				types.ErrProofClaimNotFound.Wrapf(
					"no claim found for session ID %q and supplier %q",
					unclaimedSessionHeader.GetSessionId(),
					wrongSupplierAddr,
				).Error(),
			),
		},
		{
			desc: "claim and proof session start heights must match",
			newProofMsg: func(t *testing.T) *types.MsgSubmitProof {
				// Advance the block height to shift the session start height.
				sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
				ctx = sdkCtx.WithBlockHeight(3)
				t.Cleanup(resetBlockHeightFn(&ctx))

				// Construct new proof message.
				return newTestProofMsg(t,
					supplierAddr,
					&wrongSessionStartHeightHeader,
					validSessionTree,
					expectedMerkleProofPath,
				)
			},
			expectedErr: status.Error(
				codes.FailedPrecondition,
				types.ErrProofInvalidRelay.Wrapf(
					"session headers session start heights mismatch; expected: %d, got: %d",
					2,
					1,
				).Error(),
			),
		},
		{
			desc: "claim and proof session end heights must match",
			newProofMsg: func(t *testing.T) *types.MsgSubmitProof {
				// Advance the block height such that no hydration errors can occur when
				// getting a session with start height less than the current block height.
				setBlockHeight(&ctx, 3)
				// Reset the block height to zero after this test case.
				t.Cleanup(resetBlockHeightFn(&ctx))

				// Construct new proof message.
				return newTestProofMsg(t,
					supplierAddr,
					&wrongSessionEndHeightHeader,
					validSessionTree,
					expectedMerkleProofPath,
				)
			},
			expectedErr: status.Error(
				codes.FailedPrecondition,
				types.ErrProofInvalidRelay.Wrapf(
					"session headers session end heights mismatch; expected: %d, got: %d",
					3,
					4,
				).Error(),
			),
		},
		{
			desc: "Valid proof cannot validate claim with an incorrect root",
			newProofMsg: func(t *testing.T) *types.MsgSubmitProof {
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

				// Create a claim with the incorrect Merkle root.
				wrongMerkleRootClaimMsg := newTestClaimMsg(t,
					validSessionHeader.GetSessionId(),
					supplierAddr,
					appAddr,
					service,
					wrongMerkleRootBz,
				)
				_, err = srv.CreateClaim(ctx, wrongMerkleRootClaimMsg)
				require.NoError(t, err)

				return newTestProofMsg(t,
					supplierAddr,
					validSessionHeader,
					validSessionTree,
					expectedMerkleProofPath,
				)
			},
			expectedErr: status.Error(
				codes.FailedPrecondition,
				types.ErrProofInvalidProof.Wrap("invalid closest merkle proof").Error(),
			),
		},
		{
			desc: "claim and proof application addresses must match",
			newProofMsg: func(t *testing.T) *types.MsgSubmitProof {
				t.Skip("this test case reduces to either the 'claim must exist for proof message' or 'proof session ID must match on-chain session ID cases")
				return nil
			},
		},
		{
			desc: "claim and proof service IDs must match",
			newProofMsg: func(t *testing.T) *types.MsgSubmitProof {
				t.Skip("this test case reduces to either the 'claim must exist for proof message' or 'proof session ID must match on-chain session ID cases")
				return nil
			},
		},
		{
			desc: "claim and proof supplier addresses must match",
			newProofMsg: func(t *testing.T) *types.MsgSubmitProof {
				t.Skip("this test case reduces to either the 'claim must exist for proof message' or 'proof session ID must match on-chain session ID cases")
				return nil
			},
		},
	}

	// Submit the corresponding proof.
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			proofMsg := test.newProofMsg(t)
			submitProofRes, err := srv.SubmitProof(ctx, proofMsg)

			require.ErrorContains(t, err, test.expectedErr.Error())
			require.Nil(t, submitProofRes)

			proofRes, err := keepers.AllProofs(ctx, &types.QueryAllProofsRequest{})
			require.NoError(t, err)

			// Expect zero proofs to have been persisted as all test cases are error cases.
			proofs := proofRes.GetProofs()
			require.Lenf(t, proofs, 0, "expected 0 proofs, got %d", len(proofs))
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
	sessionTree := newEmptySessionTree(t, sessionTreeHeader)

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
) relayer.SessionTree {
	t.Helper()

	// Create a temporary session tree store directory for persistence.
	testSessionTreeStoreDir, err := os.MkdirTemp("", "session_tree_store_dir")
	require.NoError(t, err)

	// Delete the temporary session tree store directory after the test completes.
	t.Cleanup(func() {
		_ = os.RemoveAll(testSessionTreeStoreDir)
	})

	// Construct a session tree to add relays to and generate a proof from.
	sessionTree, err := session.NewSessionTree(
		sessionTreeHeader,
		testSessionTreeStoreDir,
		func(*sessiontypes.SessionHeader) {},
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
		relay := newSignedEmptyRelay(
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
	sessionHeader *sessiontypes.SessionHeader,
	sessionTree relayer.SessionTree,
	closestProofPath []byte,
) *types.MsgSubmitProof {
	t.Helper()

	// Generate a closest proof from the session tree using closestProofPath.
	merkleProof, err := sessionTree.ProveClosest(closestProofPath)
	require.NoError(t, err)
	require.NotNil(t, merkleProof)

	// Serialize the closest merkle proof.
	merkleProofBz, err := merkleProof.Marshal()
	require.NoError(t, err)

	return &types.MsgSubmitProof{
		SupplierAddress: supplierAddr,
		SessionHeader:   sessionHeader,
		Proof:           merkleProofBz,
	}
}

// createClaimAndStoreBlockHash creates a valid claim, submits it on-chain,
// and on success, stores the block hash for retrieval at future heights.
// TODO_CONSIDERATION(@bryanchriswhite): Consider if we could/should split
// this into two functions.
func createClaimAndStoreBlockHash(
	ctx context.Context,
	t *testing.T,
	supplierAddr, appAddr string,
	service *sharedtypes.Service,
	sessionTree relayer.SessionTree,
	sessionHeader *sessiontypes.SessionHeader,
	msgServer types.MsgServer,
	keepers *keepertest.ProofModuleKeepers,
) {
	merkleRootBz, err := sessionTree.Flush()
	require.NoError(t, err)

	// Create a create claim message.
	claimMsg := newTestClaimMsg(t,
		sessionHeader.GetSessionId(),
		supplierAddr,
		appAddr,
		service,
		merkleRootBz,
	)
	_, err = msgServer.CreateClaim(ctx, claimMsg)
	require.NoError(t, err)

	// TODO_TECHDEBT(@red-0ne): Centralize the business logic that involves taking
	// into account the heights, windows and grace periods into helper functions.
	proofSubmissionHeight :=
		claimMsg.GetSessionHeader().GetSessionEndBlockHeight() +
			sessionkeeper.GetSessionGracePeriodBlockCount()

	// Set block height to be after the session grace period.
	blockHeightCtx := keepertest.SetBlockHeight(ctx, proofSubmissionHeight)

	// Store the current context's block hash for future height, which is currently an EndBlocker operation.
	keepers.StoreBlockHash(blockHeightCtx)
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
	relay := new(servicetypes.Relay)
	relayBz := closestMerkleProof.GetValueHash(keeper.SmtSpec)
	err = relay.Unmarshal(relayBz)
	require.NoError(t, err)

	// Retrieve the hash of the relay.
	relayHash, err := relay.GetHash()
	require.NoError(t, err)

	// Count the number of leading 0s in the relay hash to determine its difficulty.
	relayDifficultyBits, err := protocol.CountDifficultyBits(relayHash[:])
	require.NoError(t, err)

	return uint64(relayDifficultyBits)
}

// createAccount creates a new account with the given address keyring UID
// and stores it in the account keeper.
func createAccount(
	ctx context.Context,
	t *testing.T,
	addrKeyringUid string,
	keyRing keyring.Keyring,
	accountKeeper types.AccountKeeper,
) cosmostypes.AccountI {
	t.Helper()

	pubKey := createKeypair(t, addrKeyringUid, keyRing)
	addr, err := cosmostypes.AccAddressFromHexUnsafe(pubKey.Address().String())
	require.NoError(t, err)

	accountNumber := accountKeeper.NextAccountNumber(ctx)
	account := authtypes.NewBaseAccount(addr, pubKey, accountNumber, 0)
	accountKeeper.SetAccount(ctx, account)

	return account
}

// createKeypair creates a new public/private keypair that can be retrieved
// from the keyRing using the addrUid provided. It returns the corresponding
// public key.
func createKeypair(
	t *testing.T,
	addrKeyringUid string,
	keyRing keyring.Keyring,
) cryptotypes.PubKey {
	t.Helper()

	record, _, err := keyRing.NewMnemonic(
		addrKeyringUid,
		keyring.English,
		cosmostypes.FullFundraiserPath,
		keyring.DefaultBIP39Passphrase,
		hd.Secp256k1,
	)
	require.NoError(t, err)

	pubKey, err := record.GetPubKey()
	require.NoError(t, err)

	return pubKey
}

// newSignedEmptyRelay creates a new relay structure for the given req & res headers.
// It signs the relay request on behalf of application in the reqHeader.
// It signs the relay response on behalf of supplier provided..
func newSignedEmptyRelay(
	ctx context.Context,
	t *testing.T,
	supplierKeyUid, supplierAddr string,
	reqHeader, resHeader *sessiontypes.SessionHeader,
	keyRing keyring.Keyring,
	ringClient crypto.RingClient,
) *servicetypes.Relay {
	t.Helper()

	relay := newEmptyRelay(reqHeader, resHeader)
	signRelayRequest(ctx, t, relay, reqHeader.GetApplicationAddress(), keyRing, ringClient)
	signRelayResponse(ctx, t, relay, supplierKeyUid, supplierAddr, keyRing)

	return relay
}

// newEmptyRelay creates a new relay structure for the given req & res headers
// WITHOUT any payload or signatures.
func newEmptyRelay(reqHeader, resHeader *sessiontypes.SessionHeader) *servicetypes.Relay {
	return &servicetypes.Relay{
		Req: &servicetypes.RelayRequest{
			Meta: servicetypes.RelayRequestMetadata{
				SessionHeader: reqHeader,
				Signature:     nil, // Signature added elsewhere.
			},
			Payload: nil,
		},
		Res: &servicetypes.RelayResponse{
			Meta: servicetypes.RelayResponseMetadata{
				SessionHeader:     resHeader,
				SupplierSignature: nil, // Signature added elsewhere.
			},
			Payload: nil,
		},
	}
}

// TODO_TECHDEBT(@red-0ne): Centralize this logic in the relayer package.
// signRelayRequest signs the relay request (updates relay.Req.Meta.Signature)
// on behalf of appAddr using the clients provided.
func signRelayRequest(
	ctx context.Context,
	t *testing.T,
	relay *servicetypes.Relay,
	appAddr string,
	keyRing keyring.Keyring,
	ringClient crypto.RingClient,
) {
	t.Helper()

	// Retrieve the signing ring associated with the application address.
	appRing, err := ringClient.GetRingForAddress(ctx, appAddr)
	require.NoError(t, err)

	// Retrieve the signing key associated with the application address.
	signingKey := getSigningKeyFromAddress(t,
		appAddr,
		keyRing,
	)

	// Retrieve the signable bytes for the relay request.
	relayReqSignableBz, err := relay.GetReq().GetSignableBytesHash()
	require.NoError(t, err)

	// Sign the relay request.
	signature, err := appRing.Sign(relayReqSignableBz, signingKey)
	require.NoError(t, err)

	// Serialize the signature.
	signatureBz, err := signature.Serialize()
	require.NoError(t, err)

	// Update the relay request signature.
	relay.Req.Meta.Signature = signatureBz
}

// TODO_TECHDEBT(@red-0ne): Centralize this logic in the relayer package.
// in the relayer package?
// signRelayResponse signs the relay response (updates relay.Res.Meta.SupplierSignature)
// on behalf of supplierAddr using the clients provided.
func signRelayResponse(
	_ context.Context,
	t *testing.T,
	relay *servicetypes.Relay,
	supplierKeyUid, supplierAddr string,
	keyRing keyring.Keyring,
) {
	t.Helper()

	// Retrieve ths signable bytes for the relay response.
	relayResSignableBz, err := relay.GetRes().GetSignableBytesHash()
	require.NoError(t, err)

	// Sign the relay response.
	signatureBz, signerPubKey, err := keyRing.Sign(supplierKeyUid, relayResSignableBz[:], signingtypes.SignMode_SIGN_MODE_DIRECT)
	require.NoError(t, err)

	// Verify the signer address matches the expected supplier address.
	addr, err := cosmostypes.AccAddressFromBech32(supplierAddr)
	require.NoError(t, err)
	addrHexBz := strings.ToUpper(fmt.Sprintf("%x", addr.Bytes()))
	require.Equal(t, addrHexBz, signerPubKey.Address().String())

	// Update the relay response signature.
	relay.Res.Meta.SupplierSignature = signatureBz
}

// getSigningKeyFromAddress retrieves the signing key associated with the given
// bech32 address from the provided keyring.
func getSigningKeyFromAddress(t *testing.T, bech32 string, keyRing keyring.Keyring) ringtypes.Scalar {
	t.Helper()

	addr, err := cosmostypes.AccAddressFromBech32(bech32)
	require.NoError(t, err)

	armorPrivKey, err := keyRing.ExportPrivKeyArmorByAddress(addr, "")
	require.NoError(t, err)

	privKey, _, err := cosmoscrypto.UnarmorDecryptPrivKey(armorPrivKey, "")
	require.NoError(t, err)

	curve := ring_secp256k1.NewCurve()
	signingKey, err := curve.DecodeToScalar(privKey.Bytes())
	require.NoError(t, err)

	return signingKey
}

// resetBlockHeightFn returns a function that resets the block height of the
// given context to one; the first valid session block height.
func resetBlockHeightFn(ctx *context.Context) func() {
	return func() {
		setBlockHeight(ctx, 1)
	}
}

// setBlockHeight sets the block height of the given context to the given height.
func setBlockHeight(ctx *context.Context, height int64) {
	sdkCtx := cosmostypes.UnwrapSDKContext(*ctx)
	*ctx = sdkCtx.WithBlockHeight(height)
}
