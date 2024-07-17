package keeper

// TODO_TECHDEBT(@bryanchriswhite): Replace all logs in x/ from `.Info` to
// `.Debug` when the logger is replaced close to or after MainNet launch.
// Ref: https://github.com/pokt-network/poktroll/pull/448#discussion_r1549742985

import (
	"bytes"
	"context"
	"fmt"

	cosmoscryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/pokt-network/smt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/proto/types/proof"
	"github.com/pokt-network/poktroll/proto/types/service"
	"github.com/pokt-network/poktroll/proto/types/session"
	sharedtypes "github.com/pokt-network/poktroll/proto/types/shared"
	"github.com/pokt-network/poktroll/telemetry"
)

// SubmitProof is the server handler to submit and store a proof on-chain.
// A proof that's stored on-chain is what leads to rewards (i.e. inflation)
// downstream, making the series of checks a critical part of the protocol.
//
// Note: The entity sending the SubmitProof messages does not necessarily need
// to correspond to the supplier signing the proof. For example, a single entity
// could (theoretically) batch multiple proofs (signed by the corresponding supplier)
// into one transaction to save on transaction fees.
func (k msgServer) SubmitProof(
	ctx context.Context,
	msg *proof.MsgSubmitProof,
) (_ *proof.MsgSubmitProofResponse, err error) {
	// TODO_MAINNET: A potential issue with doing proof validation inside
	// `SubmitProof` is that we will not be storing false proofs on-chain (e.g. for slashing purposes).
	// This could be considered a feature (e.g. less state bloat against sybil attacks)
	// or a bug (i.e. no mechanisms for slashing suppliers who submit false proofs).
	// Revisit this prior to mainnet launch as to whether the business logic for settling sessions should be in EndBlocker or here.
	logger := k.Logger().With("method", "SubmitProof")
	logger.Info("About to start submitting proof")

	// Declare claim to reference in telemetry.
	var (
		claim           = new(proof.Claim)
		isExistingProof bool
		numRelays       uint64
		numComputeUnits uint64
	)

	// Defer telemetry calls so that they reference the final values the relevant variables.
	defer func() {
		// Only increment these metrics counters if handling a new claim.
		if !isExistingProof {
			telemetry.ClaimCounter(proof.ClaimProofStage_PROVEN, 1, err)
			telemetry.ClaimRelaysCounter(proof.ClaimProofStage_PROVEN, numRelays, err)
			telemetry.ClaimComputeUnitsCounter(proof.ClaimProofStage_PROVEN, numComputeUnits, err)
		}
	}()

	/*
		TODO_BLOCKER(@bryanchriswhite): Document these steps in proof
		verification, link to the doc for reference and delete the comments.

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
		2. verify(proof.ValueHash, expectedDifficulty); governance based
		3. verify(claim.Root, proof.ClosestProof); verify the closest proof is correct
	*/

	// Decomposing a few variables for easier access
	sessionHeader := msg.GetSessionHeader()
	supplierAddr := msg.GetSupplierAddress()

	// Helpers for logging the same metadata throughout this function calls
	logger = logger.With(
		"session_id", sessionHeader.GetSessionId(),
		"session_end_height", sessionHeader.GetSessionEndBlockHeight(),
		"supplier", supplierAddr)

	// Basic validation of the SubmitProof message.
	if err = msg.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	logger.Info("validated the submitProof message ")

	// Retrieve the supplier's public key.
	var supplierPubKey cosmoscryptotypes.PubKey
	supplierPubKey, err = k.accountQuerier.GetPubKeyFromAddress(ctx, supplierAddr)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	// Validate the session header.
	var onChainSession *session.Session
	onChainSession, err = k.queryAndValidateSessionHeader(ctx, msg)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	logger.Info("queried and validated the session header")

	// Re-hydrate message session header with the on-chain session header.
	// This corrects for discrepancies between unvalidated fields in the session header
	// which can be derived from known values (e.g. session end height).
	msg.SessionHeader = onChainSession.GetHeader()

	// Validate proof message commit height is within the respective session's
	// proof submission window using the on-chain session header.
	if err = k.validateProofWindow(ctx, msg); err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	// Unmarshal the closest merkle proof from the message.
	sparseMerkleClosestProof := &smt.SparseMerkleClosestProof{}
	if err = sparseMerkleClosestProof.Unmarshal(msg.GetProof()); err != nil {
		return nil, status.Error(codes.InvalidArgument,
			proof.ErrProofInvalidProof.Wrapf(
				"failed to unmarshal closest merkle proof: %s",
				err,
			).Error(),
		)
	}

	// TODO_MAINNET(#427): Utilize smt.VerifyCompactClosestProof here to
	// reduce on-chain storage requirements for proofs.
	// Get the relay request and response from the proof.GetClosestMerkleProof.
	relayBz := sparseMerkleClosestProof.GetValueHash(&protocol.SmtSpec)
	relay := &service.Relay{}
	if err = k.cdc.Unmarshal(relayBz, relay); err != nil {
		return nil, status.Error(
			codes.InvalidArgument,
			proof.ErrProofInvalidRelay.Wrapf(
				"failed to unmarshal relay: %s",
				err,
			).Error(),
		)
	}

	// Basic validation of the relay request.
	relayReq := relay.GetReq()
	if err = relayReq.ValidateBasic(); err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	logger.Debug("successfully validated relay request")

	// Make sure that the supplier address in the proof matches the one in the relay request.
	if supplierAddr != relayReq.Meta.SupplierAddress {
		return nil, status.Error(codes.FailedPrecondition, "supplier address mismatch")
	}
	logger.Debug("the proof supplier address matches the relay request supplier address")

	// Basic validation of the relay response.
	relayRes := relay.GetRes()
	if err = relayRes.ValidateBasic(); err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	logger.Debug("successfully validated relay response")

	// Verify that the relay request session header matches the proof session header.
	if err = compareSessionHeaders(msg.GetSessionHeader(), relayReq.Meta.GetSessionHeader()); err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	logger.Debug("successfully compared relay request session header")

	// Verify that the relay response session header matches the proof session header.
	if err = compareSessionHeaders(msg.GetSessionHeader(), relayRes.Meta.GetSessionHeader()); err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	logger.Debug("successfully compared relay response session header")

	// Verify the relay request's signature.
	if err = k.ringClient.VerifyRelayRequestSignature(ctx, relayReq); err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	logger.Debug("successfully verified relay request signature")

	// Verify the relay response's signature.
	if err = relayRes.VerifySupplierSignature(supplierPubKey); err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	logger.Debug("successfully verified relay response signature")

	// Get the proof module's governance parameters.
	params := k.GetParams(ctx)

	// Verify the relay difficulty is above the minimum required to earn rewards.
	if err = validateMiningDifficulty(relayBz, params.MinRelayDifficultyBits); err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	logger.Debug("successfully validated relay mining difficulty")

	// Validate that path the proof is submitted for matches the expected one
	// based on the pseudo-random on-chain data associated with the header.
	if err = k.validateClosestPath(
		ctx,
		sparseMerkleClosestProof,
		msg.GetSessionHeader(),
		msg.GetSupplierAddress(),
	); err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	logger.Debug("successfully validated proof path")

	// Verify the relay's difficulty.
	if err = validateMiningDifficulty(relayBz, params.MinRelayDifficultyBits); err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	// Retrieve the corresponding claim for the proof submitted so it can be
	// used in the proof validation below.
	claim, err = k.queryAndValidateClaimForProof(ctx, msg)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	logger.Debug("successfully retrieved and validated claim")

	// Verify the proof's closest merkle proof.
	if err = verifyClosestProof(sparseMerkleClosestProof, claim.GetRootHash()); err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	logger.Debug("successfully verified closest merkle proof")

	// Construct and insert newProof after all validation.
	newProof := proof.Proof{
		SupplierAddress:    supplierAddr,
		SessionHeader:      msg.GetSessionHeader(),
		ClosestMerkleProof: msg.GetProof(),
	}
	logger.Debug(fmt.Sprintf("queried and validated the claim for session ID %q", sessionHeader.SessionId))

	_, isExistingProof = k.GetProof(ctx, newProof.GetSessionHeader().GetSessionId(), newProof.GetSupplierAddress())

	k.UpsertProof(ctx, newProof)
	logger.Info("successfully upserted the proof")

	numRelays, err = claim.GetNumRelays()
	if err != nil {
		return nil, status.Error(codes.Internal, proof.ErrProofInvalidClaimRootHash.Wrap(err.Error()).Error())
	}
	numComputeUnits, err = claim.GetNumComputeUnits()
	if err != nil {
		return nil, status.Error(codes.Internal, proof.ErrProofInvalidClaimRootHash.Wrap(err.Error()).Error())
	}

	// Emit the appropriate event based on whether the claim was created or updated.
	var proofUpsertEvent proto.Message
	switch isExistingProof {
	case true:
		proofUpsertEvent = proto.Message(
			&proof.EventProofUpdated{
				Claim:           claim,
				Proof:           &newProof,
				NumRelays:       numRelays,
				NumComputeUnits: numComputeUnits,
			},
		)
	case false:
		proofUpsertEvent = proto.Message(
			&proof.EventProofSubmitted{
				Claim:           claim,
				Proof:           &newProof,
				NumRelays:       numRelays,
				NumComputeUnits: numComputeUnits,
			},
		)
	}

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	if err = sdkCtx.EventManager().EmitTypedEvent(proofUpsertEvent); err != nil {
		return nil, status.Error(
			codes.Internal,
			sharedtypes.ErrSharedEmitEvent.Wrapf(
				"failed to emit event type %T: %v",
				proofUpsertEvent,
				err,
			).Error(),
		)
	}

	return &proof.MsgSubmitProofResponse{
		Proof: &newProof,
	}, nil
}

// queryAndValidateClaimForProof ensures that a claim corresponding to the given
// proof's session exists & has a matching supplier address and session header,
// it then returns the corresponding claim if the validation is successful.
func (k msgServer) queryAndValidateClaimForProof(
	ctx context.Context,
	msg *proof.MsgSubmitProof,
) (*proof.Claim, error) {
	sessionId := msg.GetSessionHeader().GetSessionId()
	// NB: no need to assert the testSessionId or supplier address as it is retrieved
	// by respective values of the given proof. I.e., if the claim exists, then these
	// values are guaranteed to match.
	foundClaim, found := k.GetClaim(ctx, sessionId, msg.GetSupplierAddress())
	if !found {
		return nil, proof.ErrProofClaimNotFound.Wrapf(
			"no claim found for session ID %q and supplier %q",
			sessionId,
			msg.GetSupplierAddress(),
		)
	}

	claimSessionHeader := foundClaim.GetSessionHeader()
	proofSessionHeader := msg.GetSessionHeader()

	// Ensure session start heights match.
	if claimSessionHeader.GetSessionStartBlockHeight() != proofSessionHeader.GetSessionStartBlockHeight() {
		return nil, proof.ErrProofInvalidSessionStartHeight.Wrapf(
			"claim session start height %d does not match proof session start height %d",
			claimSessionHeader.GetSessionStartBlockHeight(),
			proofSessionHeader.GetSessionStartBlockHeight(),
		)
	}

	// Ensure session end heights match.
	if claimSessionHeader.GetSessionEndBlockHeight() != proofSessionHeader.GetSessionEndBlockHeight() {
		return nil, proof.ErrProofInvalidSessionEndHeight.Wrapf(
			"claim session end height %d does not match proof session end height %d",
			claimSessionHeader.GetSessionEndBlockHeight(),
			proofSessionHeader.GetSessionEndBlockHeight(),
		)
	}

	// Ensure application addresses match.
	if claimSessionHeader.GetApplicationAddress() != proofSessionHeader.GetApplicationAddress() {
		return nil, proof.ErrProofInvalidAddress.Wrapf(
			"claim application address %q does not match proof application address %q",
			claimSessionHeader.GetApplicationAddress(),
			proofSessionHeader.GetApplicationAddress(),
		)
	}

	// Ensure service IDs match.
	if claimSessionHeader.GetService().GetId() != proofSessionHeader.GetService().GetId() {
		return nil, proof.ErrProofInvalidService.Wrapf(
			"claim service ID %q does not match proof service ID %q",
			claimSessionHeader.GetService().GetId(),
			proofSessionHeader.GetService().GetId(),
		)
	}

	return &foundClaim, nil
}

// compareSessionHeaders compares a session header against an expected session header.
// This is necessary to validate the proof's session header against both the relay
// request and response's session headers.
func compareSessionHeaders(expectedSessionHeader, sessionHeader *session.SessionHeader) error {
	// Compare the Application address.
	if sessionHeader.GetApplicationAddress() != expectedSessionHeader.GetApplicationAddress() {
		return proof.ErrProofInvalidRelay.Wrapf(
			"session headers application addresses mismatch; expect: %q, got: %q",
			expectedSessionHeader.GetApplicationAddress(),
			sessionHeader.GetApplicationAddress(),
		)
	}

	// Compare the Service IDs.
	if sessionHeader.GetService().GetId() != expectedSessionHeader.GetService().GetId() {
		return proof.ErrProofInvalidRelay.Wrapf(
			"session headers service IDs mismatch; expected: %q, got: %q",
			expectedSessionHeader.GetService().GetId(),
			sessionHeader.GetService().GetId(),
		)
	}

	// Compare the Service names.
	if sessionHeader.GetService().GetName() != expectedSessionHeader.GetService().GetName() {
		return proof.ErrProofInvalidRelay.Wrapf(
			"sessionHeaders service names mismatch expect: %q, got: %q",
			expectedSessionHeader.GetService().GetName(),
			sessionHeader.GetService().GetName(),
		)
	}

	// Compare the Session start block heights.
	if sessionHeader.GetSessionStartBlockHeight() != expectedSessionHeader.GetSessionStartBlockHeight() {
		return proof.ErrProofInvalidRelay.Wrapf(
			"session headers session start heights mismatch; expected: %d, got: %d",
			expectedSessionHeader.GetSessionStartBlockHeight(),
			sessionHeader.GetSessionStartBlockHeight(),
		)
	}

	// Compare the Session end block heights.
	if sessionHeader.GetSessionEndBlockHeight() != expectedSessionHeader.GetSessionEndBlockHeight() {
		return proof.ErrProofInvalidRelay.Wrapf(
			"session headers session end heights mismatch; expected: %d, got: %d",
			expectedSessionHeader.GetSessionEndBlockHeight(),
			sessionHeader.GetSessionEndBlockHeight(),
		)
	}

	// Compare the Session IDs.
	if sessionHeader.GetSessionId() != expectedSessionHeader.GetSessionId() {
		return proof.ErrProofInvalidRelay.Wrapf(
			"session headers session IDs mismatch; expected: %q, got: %q",
			expectedSessionHeader.GetSessionId(),
			sessionHeader.GetSessionId(),
		)
	}

	return nil
}

// verifyClosestProof verifies the the correctness of the ClosestMerkleProof
// against the root hash committed to when creating the claim.
func verifyClosestProof(
	smtproof *smt.SparseMerkleClosestProof,
	claimRootHash []byte,
) error {
	valid, err := smt.VerifyClosestProof(smtproof, claimRootHash, &protocol.SmtSpec)
	if err != nil {
		return err
	}

	if !valid {
		return proof.ErrProofInvalidProof.Wrap("invalid closest merkle proof")
	}

	return nil
}

// validateMiningDifficulty ensures that the relay's mining difficulty meets the
// required minimum threshold.
// TODO_TECHDEBT: Factor out the relay mining difficulty validation into a shared
// function that can be used by both the proof and the miner packages.
func validateMiningDifficulty(relayBz []byte, minRelayDifficultyBits uint64) error {
	relayHash := service.GetHashFromBytes(relayBz)
	relayDifficultyBits := protocol.CountHashDifficultyBits(relayHash)

	// TODO_MAINNET: Devise a test that tries to attack the network and ensure that there
	// is sufficient telemetry.
	if uint64(relayDifficultyBits) < minRelayDifficultyBits {
		return proof.ErrProofInvalidRelay.Wrapf(
			"relay difficulty %d is less than the minimum difficulty %d",
			relayDifficultyBits,
			minRelayDifficultyBits,
		)
	}

	return nil
}

// validateClosestPath ensures that the proof's path matches the expected path.
// Since the proof path needs to be pseudo-randomly selected AFTER the session
// ends, the seed for this is the block hash at the height when the proof window
// opens.
func (k msgServer) validateClosestPath(
	ctx context.Context,
	smtProof *smt.SparseMerkleClosestProof,
	sessionHeader *session.SessionHeader,
	supplierAddr string,
) error {
	// The RelayMiner has to wait until the submit claim and proof windows is are open
	// in order to to create the claim and submit claims and proofs, respectively.
	// These windows are calculated as specified in the docs;
	// see: https://dev.poktroll.com/protocol/primitives/claim_and_proof_lifecycle.
	//
	// For reference, see relayerSessionsManager#waitForEarliest{CreateClaim,SubmitProof}Height().
	//
	// The RelayMiner has to wait this long to ensure that late relays (i.e.
	// submitted during SessionNumber=(N+1) but created during SessionNumber=N) are
	// still included as part of SessionNumber=N.
	//
	// Since smt.ProveClosest is defined in terms of proof window open height,
	// this block's hash needs to be used for validation too.
	earliestSupplierProofCommitHeight, err := k.sharedQuerier.GetEarliestSupplierProofCommitHeight(
		ctx,
		sessionHeader.GetSessionEndBlockHeight(),
		supplierAddr,
	)
	if err != nil {
		return err
	}

	// earliestSupplierProofCommitHeight - 1 is the block that will have its hash used as the
	// source of entropy for all the session trees in that batch, waiting for it to
	// be received before proceeding.
	proofPathSeedBlockHash := k.sessionKeeper.GetBlockHash(ctx, earliestSupplierProofCommitHeight-1)

	// TODO_BETA: Investigate "proof for the path provided does not match one expected by the on-chain protocol"
	// error that may occur due to block height differing from the off-chain part.
	k.logger.Info("E2E_DEBUG: height for block hash when verifying the proof", earliestSupplierProofCommitHeight, sessionHeader.GetSessionId())

	expectedProofPath := protocol.GetPathForProof(proofPathSeedBlockHash, sessionHeader.GetSessionId())
	if !bytes.Equal(smtProof.Path, expectedProofPath) {
		return proof.ErrProofInvalidProof.Wrapf(
			"the path of the proof provided (%x) does not match one expected by the on-chain protocol (%x)",
			smtProof.Path,
			expectedProofPath,
		)
	}

	return nil
}
