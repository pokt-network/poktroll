package keeper

import (
	"bytes"
	"context"
	"crypto/sha256"

	"github.com/pokt-network/smt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/pkg/relayer/protocol"
	"github.com/pokt-network/poktroll/x/proof/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

const (
	// relayMinDifficultyBits is the minimum difficulty that a relay must have to be
	// volume / reward applicable.
	// TODO_BLOCKER: relayMinDifficultyBits should be a governance-based parameter
	relayMinDifficultyBits = 0

	// sumSize is the size of the sum of the relay request and response
	// in bytes. This is used to extract the relay request and response
	// from the closest merkle proof.
	// TODO_TECHDEBT: Have a method on the smst to extract the value hash or export
	// sumSize to be used instead of current local value
	sumSize = 8
)

// SMT specification used for the proof verification.
var spec *smt.TrieSpec

func init() {
	// Use a no prehash spec that returns a nil value hasher for the proof
	// verification to avoid hashing the value twice.
	spec = smt.NoPrehashSpec(sha256.New(), true)
}

func (k msgServer) SubmitProof(ctx context.Context, msg *types.MsgSubmitProof) (*types.MsgSubmitProofResponse, error) {
	// TODO_BLOCKER: Prevent Proof upserts after the tokenomics module has processes the respective session.
	logger := k.Logger().With("TECHDEBTmethod", "SubmitProof")
	logger.Debug("submitting proof")

	/*
		TODO_INCOMPLETE: Handling the message

		## Actions (error if anything fails)
		1. Retrieve a fully hydrated `session` from on-chain store using `msg` metadata
		2. Retrieve a fully hydrated `claim` from on-chain store using `msg` metadata
		3. Retrieve `relay.Req` and `relay.Res` from deserializing `proof.ClosestValueHash`

		## Basic Validations (metadata only)
		1. proof.sessionId == claim.sessionId
		2. msg.supplier in session.suppliers
		3. relay.Req.signer == session.appAddr
		4. relay.Res.signer == msg.supplier

		## Msg distribution validation (governance based params)
		1. Validate Proof submission is not too early; governance-based param + pseudo-random variation
		2. Validate Proof submission is not too late; governance-based param + pseudo-random variation

		## Relay Signature validation
		1. verify(relay.Req.Signature, appRing)
		2. verify(relay.Res.Signature, supplier.pubKey)

		## Relay Mining validation
		1. verify(proof.path) is the expected path; pseudo-random variation using on-chain data
		2. verify(proof.ValueHash, expectedDiffictulty); governance based
		3. verify(claim.Root, proof.ClosestProof); verify the closest proof is correct
	*/

	// Ensure that all validation and verification checks are successful on the
	// MsgSubmitProof message before constructing the proof and inserting it into
	// the store.

	if err := msg.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	supplierPubKey, err := k.pubKeyClient.GetPubKeyFromAddress(ctx, msg.GetSupplierAddress())
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	if _, err := k.queryAndValidateSessionHeader(
		ctx,
		msg.GetSessionHeader(),
		msg.GetSupplierAddress(),
	); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Unmarshal the closest merkle proof from the message.
	sparseMerkleClosestProof := &smt.SparseMerkleClosestProof{}
	if err := sparseMerkleClosestProof.Unmarshal(msg.GetProof()); err != nil {
		return nil, types.ErrProofInvalidProof.Wrapf(
			"failed to unmarshal closest merkle proof: %s",
			err,
		)
	}

	// Get the relay request and response from the proof.GetClosestMerkleProof.
	closestValueHash := sparseMerkleClosestProof.ClosestValueHash
	relayBz := closestValueHash[:len(closestValueHash)-sumSize]
	relay := &servicetypes.Relay{}
	if err := k.cdc.Unmarshal(relayBz, relay); err != nil {
		return nil, types.ErrProofInvalidRelay.Wrapf(
			"failed to unmarshal relay: %s",
			err,
		)
	}

	if err := relay.GetReq().ValidateBasic(); err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	if err := relay.GetRes().ValidateBasic(); err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	// Verify that the relay request session header matches the proof session header.
	if err := compareSessionHeaders(msg.GetSessionHeader(), relay.GetReq().Meta.GetSessionHeader()); err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	// Verify that the relay response session header matches the proof session header.
	if err := compareSessionHeaders(msg.GetSessionHeader(), relay.GetRes().Meta.GetSessionHeader()); err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	// Verify the relay response's signature.
	if err := relay.GetRes().VerifySupplierSignature(supplierPubKey); err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	// Verify the relay request's signature.
	if err := k.ringClient.VerifyRelayRequestSignature(ctx, relay.GetReq()); err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	// Validate that proof's path matches the earliest proof submission block hash.
	if err := k.validateClosestPath(ctx, sparseMerkleClosestProof, msg.GetSessionHeader()); err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	// Verify the relay's difficulty.
	if err := validateMiningDifficulty(relayBz); err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	claim, err := k.queryAndValidateClaimForProof(ctx, msg)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	// Verify the proof's closest merkle proof.
	if err := verifyClosestProof(sparseMerkleClosestProof, claim.GetRootHash()); err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	// Construct and insert proof after all validation.
	proof := types.Proof{
		SupplierAddress:    msg.GetSupplierAddress(),
		SessionHeader:      msg.GetSessionHeader(),
		ClosestMerkleProof: msg.GetProof(),
	}

	// TODO_BLOCKER: check if this proof already exists and return an appropriate error
	// in any case where the supplier should no longer be able to update the given proof.
	k.Keeper.UpsertProof(ctx, proof)

	// TODO_UPNEXT(@Olshansk, #359): Call `tokenomics.SettleSessionAccounting()` here

	logger.
		With(
			"session_id", proof.GetSessionHeader().GetSessionId(),
			"session_end_height", proof.GetSessionHeader().GetSessionEndBlockHeight(),
			"supplier", proof.GetSupplierAddress(),
		).
		Debug("created proof")

	return &types.MsgSubmitProofResponse{}, nil
}

// queryAndValidateClaimForProof ensures that a claim corresponding to the given
// proof's session exists & has a matching supplier address and session header,
// it then returns the corresponding claim if the validation is successful.
func (k msgServer) queryAndValidateClaimForProof(
	ctx context.Context,
	msg *types.MsgSubmitProof,
) (*types.Claim, error) {
	sessionId := msg.GetSessionHeader().GetSessionId()
	// NB: no need to assert the testSessionId or supplier address as it is retrieved
	// by respective values of the given proof. I.e., if the claim exists, then these
	// values are guaranteed to match.
	foundClaim, found := k.GetClaim(ctx, sessionId, msg.GetSupplierAddress())
	if !found {
		return nil, types.ErrProofClaimNotFound.Wrapf(
			"no claim found for session ID %q and supplier %q",
			sessionId, msg.GetSupplierAddress(),
		)
	}

	claimSessionHeader := foundClaim.GetSessionHeader()
	proofSessionHeader := msg.GetSessionHeader()

	// Ensure session start heights match.
	if claimSessionHeader.GetSessionStartBlockHeight() != proofSessionHeader.GetSessionStartBlockHeight() {
		return nil, types.ErrProofInvalidSessionStartHeight.Wrapf(
			"claim session start height %d does not match proof session start height %d",
			claimSessionHeader.GetSessionStartBlockHeight(),
			proofSessionHeader.GetSessionStartBlockHeight(),
		)
	}

	// Ensure session end heights match.
	if claimSessionHeader.GetSessionEndBlockHeight() != proofSessionHeader.GetSessionEndBlockHeight() {
		return nil, types.ErrProofInvalidSessionEndHeight.Wrapf(
			"claim session end height %d does not match proof session end height %d",
			claimSessionHeader.GetSessionEndBlockHeight(),
			proofSessionHeader.GetSessionEndBlockHeight(),
		)
	}

	// Ensure application addresses match.
	if claimSessionHeader.GetApplicationAddress() != proofSessionHeader.GetApplicationAddress() {
		return nil, types.ErrProofInvalidAddress.Wrapf(
			"claim application address %q does not match proof application address %q",
			claimSessionHeader.GetApplicationAddress(),
			proofSessionHeader.GetApplicationAddress(),
		)
	}

	// Ensure service IDs match.
	if claimSessionHeader.GetService().GetId() != proofSessionHeader.GetService().GetId() {
		return nil, types.ErrProofInvalidService.Wrapf(
			"claim service ID %q does not match proof service ID %q",
			claimSessionHeader.GetService().GetId(),
			proofSessionHeader.GetService().GetId(),
		)
	}

	return &foundClaim, nil
}

// compareSessionHeaders compares a session header against an expected session header.
func compareSessionHeaders(expectedSessionHeader, sessionHeader *sessiontypes.SessionHeader) error {
	if sessionHeader.GetApplicationAddress() != expectedSessionHeader.GetApplicationAddress() {
		return types.ErrProofInvalidRelay.Wrapf(
			"sessionHeaders application addresses mismatch expect: %q, got: %q",
			expectedSessionHeader.GetApplicationAddress(),
			sessionHeader.GetApplicationAddress(),
		)
	}

	if sessionHeader.GetService().GetId() != expectedSessionHeader.GetService().GetId() {
		return types.ErrProofInvalidRelay.Wrapf(
			"sessionHeaders service IDs mismatch expect: %q, got: %q",
			expectedSessionHeader.GetService().GetId(),
			sessionHeader.GetService().GetId(),
		)
	}

	if sessionHeader.GetService().GetName() != expectedSessionHeader.GetService().GetName() {
		return types.ErrProofInvalidRelay.Wrapf(
			"sessionHeaders service names mismatch expect: %q, got: %q",
			expectedSessionHeader.GetService().GetName(),
			sessionHeader.GetService().GetName(),
		)
	}

	if sessionHeader.GetSessionStartBlockHeight() != expectedSessionHeader.GetSessionStartBlockHeight() {
		return types.ErrProofInvalidRelay.Wrapf(
			"sessionHeaders session start heights mismatch expect: %d, got: %d",
			expectedSessionHeader.GetSessionStartBlockHeight(),
			sessionHeader.GetSessionStartBlockHeight(),
		)
	}

	if sessionHeader.GetSessionEndBlockHeight() != expectedSessionHeader.GetSessionEndBlockHeight() {
		return types.ErrProofInvalidRelay.Wrapf(
			"sessionHeaders session end heights mismatch expect: %d, got: %d",
			expectedSessionHeader.GetSessionEndBlockHeight(),
			sessionHeader.GetSessionEndBlockHeight(),
		)
	}

	if sessionHeader.GetSessionId() != expectedSessionHeader.GetSessionId() {
		return types.ErrProofInvalidRelay.Wrapf(
			"sessionHeaders session IDs mismatch expect: %q, got: %q",
			expectedSessionHeader.GetSessionId(),
			sessionHeader.GetSessionId(),
		)
	}

	return nil
}

// verifyClosestProof verifies the closest merkle proof against the expected root hash.
func verifyClosestProof(
	proof *smt.SparseMerkleClosestProof,
	expectedRootHash []byte,
) error {
	valid, err := smt.VerifyClosestProof(proof, expectedRootHash, spec)
	if err != nil {
		return err
	}

	if !valid {
		return types.ErrProofInvalidProof.Wrap("invalid closest merkle proof")
	}

	return nil
}

// validateMiningDifficulty ensures that the relay's mining difficulty meets the required
// difficulty.
// TODO_TECHDEBT: Factor out the relay mining difficulty validation into a shared function
// that can be used by both the proof and the miner packages.
func validateMiningDifficulty(relayBz []byte) error {
	hasher := sha256.New()
	hasher.Write(relayBz)
	relayHash := hasher.Sum(nil)

	difficultyBits, err := protocol.CountDifficultyBits(relayHash)
	if err != nil {
		return types.ErrProofInvalidRelay.Wrapf(
			"error counting difficulty bits: %s",
			err,
		)
	}

	// TODO: Devise a test that tries to attack the network and ensure that there
	// is sufficient telemetry.
	if difficultyBits < relayMinDifficultyBits {
		return types.ErrProofInvalidRelay.Wrapf(
			"relay difficulty %d is less than the required difficulty %d",
			difficultyBits,
			relayMinDifficultyBits,
		)
	}

	return nil
}

// validateClosestPath ensures that the proof's path matches the expected path.
func (k msgServer) validateClosestPath(
	ctx context.Context,
	proof *smt.SparseMerkleClosestProof,
	sessionHeader *sessiontypes.SessionHeader,
) error {
	// The RelayMiner has to wait until the createClaimWindowStartHeight and the
	// submitProofWindowStartHeight are open to respectively create the claim and
	// submit the proof respectively.
	// These windows are calculated as (SessionEndBlockHeight + GracePeriodBlockCount).
	// For reference, see relayerSessionsManager.waitForEarliest{CreateClaim,SubmitProof}Height().
	// The RelayMiner has to wait this long to ensure that late relays (i.e.
	// submitted during SessionNumber=(N+1) but created during SessionNumber=N) are
	// still included as part of SessionNumber=N.
	// Since smt.ProveClosest is defined in terms of submitProofWindowStartHeight,
	// this block's hash needs to be used for validation too.
	// TODO_TECHDEBT(#409): Reference the session rollover documentation here.
	sessionEndWithGracePeriodBlockHeight := sessionHeader.GetSessionEndBlockHeight() +
		sessionkeeper.GetSessionGracePeriodBlockCount()
	blockHash := k.sessionKeeper.GetBlockHash(ctx, sessionEndWithGracePeriodBlockHeight)

	if !bytes.Equal(proof.Path, blockHash) {
		return types.ErrProofInvalidProof.Wrapf(
			"proof path %x does not match block hash %x",
			proof.Path,
			blockHash,
		)
	}

	return nil
}
