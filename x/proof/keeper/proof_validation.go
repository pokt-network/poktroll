package keeper

/*
	TODO_MAINNET: Document these steps in the docs and link here.

	## Actions (error if anything fails)
	1. Retrieve a fully hydrated `session` from onchain store using `msg` metadata
	2. Retrieve a fully hydrated `claim` from onchain store using `msg` metadata
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
	1. verify(proof.path) is the expected path; pseudo-random variation using onchain data
	2. verify(proof.ValueHash, expectedDifficulty); governance based
	3. verify(claim.Root, proof.ClosestProof); verify the closest proof is correct
*/

import (
	"bytes"
	"context"
	"fmt"

	cosmostelemetry "github.com/cosmos/cosmos-sdk/telemetry"
	"github.com/pokt-network/smt"

	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/telemetry"
	"github.com/pokt-network/poktroll/x/proof/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

// EnsureWellFormedProof validates a supplier's proof for:
//  1. Valid session header
//  2. Submission height within window
//  3. Matching relay request/response headers
//  4. Relay Mining difficulty above reward threshold
//
// EnsureWellFormedProof does not validate computationally expensive operations like:
//  1. Proof relay signatures
//  2. ClosestMerkleProof
//
// Additional developer context as of #1031:
//   - This function is expected to be called from the SubmitProof messages handler
//   - Computationally expensive operations are left to the block's EndBlocker
//
// NOTE: Full validation requires passing both:
//  1. EnsureWellFormedProof (this function)
//  2. EnsureValidProofSignaturesAndClosestPath
func (k Keeper) EnsureWellFormedProof(ctx context.Context, proof *types.Proof) error {
	logger := k.Logger().With("method", "EnsureWellFormedProof")

	supplierOperatorAddr := proof.SupplierOperatorAddress

	// Validate the session header.
	var onChainSession *sessiontypes.Session
	onChainSession, err := k.queryAndValidateSessionHeader(ctx, proof.SessionHeader, supplierOperatorAddr)
	if err != nil {
		return err
	}
	logger.Info("queried and validated the session header")

	// Re-hydrate message session header with the onchain session header.
	// This corrects for discrepancies between unvalidated fields in the session
	// header which can be derived from known values (e.g. session end height).
	sessionHeader := onChainSession.GetHeader()

	logger = logger.With(
		"session_id", sessionHeader.GetSessionId(),
		"application_address", sessionHeader.GetApplicationAddress(),
		"service_id", sessionHeader.GetServiceId(),
		"session_end_height", sessionHeader.GetSessionEndBlockHeight(),
		"supplier_operator_address", supplierOperatorAddr,
	)

	// Validate proof message commit height is within the respective session's
	// proof submission window using the onchain session header.
	if err = k.validateProofWindow(ctx, sessionHeader, supplierOperatorAddr); err != nil {
		logger.Error(fmt.Sprintf("failed to validate proof window due to error: %v", err))
		return err
	}

	if len(proof.ClosestMerkleProof) == 0 {
		logger.Error("closest merkle proof cannot be empty")
		return types.ErrProofInvalidProof.Wrap("closest merkle proof cannot be empty")
	}

	// Unmarshal the sparse compact closest merkle proof from the message.
	sparseCompactMerkleClosestProof := &smt.SparseCompactMerkleClosestProof{}
	if err = sparseCompactMerkleClosestProof.Unmarshal(proof.ClosestMerkleProof); err != nil {
		logger.Error(fmt.Sprintf("failed to unmarshal sparse compact merkle closest proof due to error: %v", err))
		return types.ErrProofInvalidProof.Wrapf("failed to unmarshal sparse compact merkle closest proof: %s", err)
	}

	// Get the relay bytes from the proof to validate the relay request and response.
	// Checking that the relayBz hash matches the SMST's closest value hash is done
	// in the EnsureValidProofSignaturesAndClosestPath function.
	relayBz := proof.GetProofRelay()
	relay := &servicetypes.Relay{}
	if err = k.cdc.Unmarshal(relayBz, relay); err != nil {
		logger.Error(fmt.Sprintf("failed to unmarshal relay due to error: %v", err))
		return types.ErrProofInvalidRelay.Wrapf("failed to unmarshal relay: %s", err)
	}

	// Basic validation of the relay request.
	relayReq := relay.GetReq()
	if err = relayReq.ValidateBasic(); err != nil {
		logger.Error(fmt.Sprintf("failed to validate relay request due to error: %v", err))
		return err
	}
	logger.Debug("successfully validated relay request")

	// Make sure that the supplier operator address in the proof matches the one in the relay request.
	if supplierOperatorAddr != relayReq.Meta.SupplierOperatorAddress {
		logger.Error(fmt.Sprintf(
			"supplier operator address mismatch; proof: %s, relay request: %s",
			supplierOperatorAddr,
			relayReq.Meta.SupplierOperatorAddress,
		))
		return types.ErrProofSupplierMismatch.Wrapf("supplier type mismatch")
	}
	logger.Debug("the proof supplier operator address matches the relay request supplier operator address")

	// Basic validation of the relay response.
	relayRes := relay.GetRes()
	if err = relayRes.ValidateBasic(); err != nil {
		logger.Error(fmt.Sprintf("failed to validate relay response due to error: %v", err))
		return err
	}
	logger.Debug("successfully validated relay response")

	// Verify that the relay request session header matches the proof session header.
	if err = compareSessionHeaders(sessionHeader, relayReq.Meta.GetSessionHeader()); err != nil {
		logger.Error(fmt.Sprintf("relay request and proof session header mismatch: %v", err))
		return err
	}
	logger.Debug("successfully compared relay request session header")

	// Verify that the relay response session header matches the proof session header.
	if err = compareSessionHeaders(sessionHeader, relayRes.Meta.GetSessionHeader()); err != nil {
		logger.Error(fmt.Sprintf("relay response and proof session header mismatch: %v", err))
		return err
	}
	logger.Debug("successfully compared relay response session header")

	// Get the service's relay mining difficulty.
	serviceRelayDifficulty, _ := k.serviceKeeper.GetRelayMiningDifficulty(ctx, sessionHeader.GetServiceId())

	// Verify the relay difficulty is above the minimum required to earn rewards.
	if err = validateRelayDifficulty(
		relayBz,
		serviceRelayDifficulty.GetTargetHash(),
	); err != nil {
		logger.Error(fmt.Sprintf("failed to validate relay difficulty due to error: %v", err))
		return types.ErrProofInvalidRelayDifficulty.Wrapf("failed to validate relay difficulty for service %s due to: %v", sessionHeader.ServiceId, err)
	}
	logger.Debug("successfully validated relay mining difficulty")

	// Retrieve the corresponding claim for the proof submitted
	if err := k.validateSessionClaim(ctx, sessionHeader, supplierOperatorAddr); err != nil {
		return err
	}
	logger.Debug("successfully retrieved and validated claim")

	return nil
}

// EnsureValidProofSignaturesAndClosestPath validates:
//  1. Proof signatures from the supplier
//  2. Valid relay request/response signatures from the application/supplier respectively
//  3. Closest path validation against onchain claim
//
// Execution requirements:
//  1. Must run in the EndBlocker of the proof submission height
//  2. Cannot run during SubmitProof due to computational cost
//
// NOTE: Full validation requires passing both:
//  1. EnsureWellFormedProof
//  2. EnsureValidProofSignaturesAndClosestPath (this function)
func (k Keeper) EnsureValidProofSignaturesAndClosestPath(
	ctx context.Context,
	claim *types.Claim,
	proof *types.Proof,
) error {
	// Telemetry: measure execution time.
	defer cosmostelemetry.MeasureSince(cosmostelemetry.Now(), telemetry.MetricNameKeys("proof", "validation")...)

	sessionHeader := proof.GetSessionHeader()
	supplierOperatorAddr := proof.SupplierOperatorAddress

	logger := k.Logger().With(
		"method", "EnsureValidProofSignaturesAndClosestPath",
		"session_id", sessionHeader.GetSessionId(),
		"application_address", sessionHeader.GetApplicationAddress(),
		"service_id", sessionHeader.GetServiceId(),
		"session_end_height", sessionHeader.GetSessionEndBlockHeight(),
		"supplier_operator_address", supplierOperatorAddr,
	)

	// Retrieve the supplier operator's public key.
	supplierOperatorPubKey, err := k.accountQuerier.GetPubKeyFromAddress(ctx, supplierOperatorAddr)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to retrieve supplier operator public key due to error: %v", err))
		return err
	}

	// Unmarshal the sparse compact merkle closest proof from the message.
	sparseCompactMerkleClosestProof := &smt.SparseCompactMerkleClosestProof{}
	if err = sparseCompactMerkleClosestProof.Unmarshal(proof.ClosestMerkleProof); err != nil {
		logger.Error(fmt.Sprintf("failed to unmarshal sparse compact merkle closest proof due to error: %v", err))
		return types.ErrProofInvalidProof.Wrapf("failed to unmarshal sparse compact merkle closest proof: %s", err)
	}

	// SparseCompactMerkeClosestProof was intentionally compacted to reduce its onchain state size
	// so it must be decompacted rather than just retrieving the value via GetValueHash (not implemented).
	sparseMerkleClosestProof, err := smt.DecompactClosestProof(sparseCompactMerkleClosestProof, protocol.NewSMTSpec())
	if err != nil {
		logger.Error(fmt.Sprintf("failed to decompact sparse merkle closest proof due to error: %v", err))
		return types.ErrProofInvalidProof.Wrapf("failed to decompact sparse merkle closest proof: %s", err)
	}

	// Get the proof value hash from the proof.GetClosestMerkleProof.
	proofValueHash := sparseMerkleClosestProof.GetValueHash(protocol.NewSMTSpec())

	// Validate the relay generated hash is the same as the one extracted from the proof.
	relayBz := proof.GetProofRelay()
	relayBzHash := protocol.GetRelayHashFromBytes(relayBz)
	if !bytes.Equal(proofValueHash, relayBzHash[:]) {
		logger.Error(fmt.Sprintf(
			"relay hash mismatch; proof: %x, proof relay: %x",
			proofValueHash,
			relayBzHash,
		))
		return types.ErrProofInvalidRelay.Wrap("relay hash mismatch")
	}

	relay := &servicetypes.Relay{}
	if err = k.cdc.Unmarshal(relayBz, relay); err != nil {
		logger.Error(fmt.Sprintf("failed to unmarshal relay due to error: %v", err))
		return types.ErrProofInvalidRelay.Wrapf("failed to unmarshal relay: %s", err)
	}

	// Verify the relay request's signature.
	if err = k.ringClient.VerifyRelayRequestSignature(ctx, relay.GetReq()); err != nil {
		logger.Error(fmt.Sprintf("failed to verify relay request signature due to error: %v", err))
		return err
	}
	logger.Debug("successfully verified relay request signature")

	// Verify the relay response's signature.
	if err = relay.GetRes().VerifySupplierOperatorSignature(supplierOperatorPubKey); err != nil {
		logger.Error(fmt.Sprintf("failed to verify relay response signature due to error: %v", err))
		return err
	}
	logger.Debug("successfully verified relay response signature")

	// Validate that path the proof is submitted for matches the expected one
	// based on the pseudo-random onchain data associated with the header.
	if err = k.validateClosestPath(
		ctx,
		sparseMerkleClosestProof,
		sessionHeader,
		supplierOperatorAddr,
	); err != nil {
		logger.Error(fmt.Sprintf("failed to validate closest path due to error: %v", err))
		return err
	}
	logger.Debug("successfully validated proof path")

	// Verify the proof's sparse merkle closest proof.
	if err = verifyClosestProof(sparseMerkleClosestProof, claim.GetRootHash()); err != nil {
		logger.Error(fmt.Sprintf("failed to verify sparse merkle closest proof due to error: %v", err))
		return err
	}
	logger.Debug("successfully verified sparse merkle closest proof")

	return nil
}

// validateClosestPath ensures that the proof's path matches the expected path.
// Since the proof path needs to be pseudo-randomly selected AFTER the session
// ends, the seed for this is the block hash at the height when the proof window
// opens.
func (k Keeper) validateClosestPath(
	ctx context.Context,
	proof *smt.SparseMerkleClosestProof,
	sessionHeader *sessiontypes.SessionHeader,
	supplierOperatorAddr string,
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
		supplierOperatorAddr,
	)
	if err != nil {
		return err
	}

	// earliestSupplierProofCommitHeight - 1 is the block that will have its hash used as the
	// source of entropy for all the session trees in that batch, waiting for it to
	// be received before proceeding.
	proofPathSeedBlockHash := k.sessionKeeper.GetBlockHash(ctx, earliestSupplierProofCommitHeight-1)

	expectedProofPath := protocol.GetPathForProof(proofPathSeedBlockHash, sessionHeader.GetSessionId())
	if !bytes.Equal(proof.Path, expectedProofPath) {
		return types.ErrProofInvalidProof.Wrapf(
			"the path of the proof provided (%x) does not match one expected by the onchain protocol (%x)",
			proof.Path,
			expectedProofPath,
		)
	}

	return nil
}

// validateSessionClaim ensures that the given session header and supplierOperatorAddress
// have a corresponding claim.
func (k Keeper) validateSessionClaim(
	ctx context.Context,
	sessionHeader *sessiontypes.SessionHeader,
	supplierOperatorAddr string,
) error {
	sessionId := sessionHeader.SessionId

	// Retrieve the claim corresponding to the session ID and supplier operator address.
	foundClaim, found := k.GetClaim(ctx, sessionId, supplierOperatorAddr)
	if !found {
		return types.ErrProofClaimNotFound.Wrapf(
			"no claim found for session ID %q and supplier %q",
			sessionId,
			supplierOperatorAddr,
		)
	}

	claimSessionHeader := foundClaim.GetSessionHeader()

	// Ensure session start heights match.
	if claimSessionHeader.GetSessionStartBlockHeight() != sessionHeader.GetSessionStartBlockHeight() {
		return types.ErrProofInvalidSessionStartHeight.Wrapf(
			"claim session start height %d does not match proof session start height %d",
			claimSessionHeader.GetSessionStartBlockHeight(),
			sessionHeader.GetSessionStartBlockHeight(),
		)
	}

	// Ensure session end heights match.
	if claimSessionHeader.GetSessionEndBlockHeight() != sessionHeader.GetSessionEndBlockHeight() {
		return types.ErrProofInvalidSessionEndHeight.Wrapf(
			"claim session end height %d does not match proof session end height %d",
			claimSessionHeader.GetSessionEndBlockHeight(),
			sessionHeader.GetSessionEndBlockHeight(),
		)
	}

	// Ensure application addresses match.
	if claimSessionHeader.GetApplicationAddress() != sessionHeader.GetApplicationAddress() {
		return types.ErrProofInvalidAddress.Wrapf(
			"claim application address %q does not match proof application address %q",
			claimSessionHeader.GetApplicationAddress(),
			sessionHeader.GetApplicationAddress(),
		)
	}

	// Ensure service IDs match.
	if claimSessionHeader.GetServiceId() != sessionHeader.GetServiceId() {
		return types.ErrProofInvalidService.Wrapf(
			"claim service ID %q does not match proof service ID %q",
			claimSessionHeader.GetServiceId(),
			sessionHeader.GetServiceId(),
		)
	}

	return nil
}

// compareSessionHeaders compares a session header against an expected session header.
// This is necessary to validate the proof's session header against both the relay
// request and response's session headers.
func compareSessionHeaders(expectedSessionHeader, sessionHeader *sessiontypes.SessionHeader) error {
	// Compare the Application address.
	if sessionHeader.GetApplicationAddress() != expectedSessionHeader.GetApplicationAddress() {
		return types.ErrProofInvalidRelay.Wrapf(
			"session headers application addresses mismatch; expect: %q, got: %q",
			expectedSessionHeader.GetApplicationAddress(),
			sessionHeader.GetApplicationAddress(),
		)
	}

	// Compare the Service IDs.
	if sessionHeader.GetServiceId() != expectedSessionHeader.GetServiceId() {
		return types.ErrProofInvalidRelay.Wrapf(
			"session headers service IDs mismatch; expected: %q, got: %q",
			expectedSessionHeader.GetServiceId(),
			sessionHeader.GetServiceId(),
		)
	}

	// Compare the Session start block heights.
	if sessionHeader.GetSessionStartBlockHeight() != expectedSessionHeader.GetSessionStartBlockHeight() {
		return types.ErrProofInvalidRelay.Wrapf(
			"session headers session start heights mismatch; expected: %d, got: %d",
			expectedSessionHeader.GetSessionStartBlockHeight(),
			sessionHeader.GetSessionStartBlockHeight(),
		)
	}

	// Compare the Session end block heights.
	if sessionHeader.GetSessionEndBlockHeight() != expectedSessionHeader.GetSessionEndBlockHeight() {
		return types.ErrProofInvalidRelay.Wrapf(
			"session headers session end heights mismatch; expected: %d, got: %d",
			expectedSessionHeader.GetSessionEndBlockHeight(),
			sessionHeader.GetSessionEndBlockHeight(),
		)
	}

	// Compare the Session IDs.
	if sessionHeader.GetSessionId() != expectedSessionHeader.GetSessionId() {
		return types.ErrProofInvalidRelay.Wrapf(
			"session headers session IDs mismatch; expected: %q, got: %q",
			expectedSessionHeader.GetSessionId(),
			sessionHeader.GetSessionId(),
		)
	}

	return nil
}

// verifyClosestProof verifies the correctness of the ClosestMerkleProof
// against the root hash committed to when creating the claim.
func verifyClosestProof(
	proof *smt.SparseMerkleClosestProof,
	claimRootHash []byte,
) error {
	valid, err := smt.VerifyClosestProof(proof, claimRootHash, protocol.NewSMTSpec())
	if err != nil {
		return err
	}

	if !valid {
		return types.ErrProofInvalidProof.Wrap("invalid closest merkle proof")
	}

	return nil
}

// validateRelayDifficulty ensures that the relay's mining difficulty meets the
// required minimum difficulty of the service.
// TODO_TECHDEBT(@red-0ne): Factor out to the relay mining difficulty validation into a shared
// function that can be used by both the proof and the miner packages.
func validateRelayDifficulty(relayBz, serviceRelayDifficultyTargetHash []byte) error {
	// This should theoretically never happen, but it's better to be safe than sorry.
	if len(serviceRelayDifficultyTargetHash) != protocol.RelayHasherSize {
		return types.ErrProofInvalidRelay.Wrapf(
			"invalid RelayDifficultyTargetHash: (%x); length wanted: %d; got: %d",
			serviceRelayDifficultyTargetHash,
			protocol.RelayHasherSize,
			len(serviceRelayDifficultyTargetHash),
		)
	}

	// Convert the array to a slice
	relayHashArr := protocol.GetRelayHashFromBytes(relayBz)
	relayHash := relayHashArr[:]

	// Relay difficulty is within the service difficulty
	if protocol.IsRelayVolumeApplicable(relayHash, serviceRelayDifficultyTargetHash) {
		return nil
	}

	relayDifficultyMultiplierStr := protocol.GetRelayDifficultyMultiplier(relayHash).String()
	targetDifficultyMultiplierStr := protocol.GetRelayDifficultyMultiplier(serviceRelayDifficultyTargetHash).String()

	return types.ErrProofInvalidRelay.Wrapf(
		"the difficulty relay being proven is (%s), and is smaller than the target difficulty (%s)",
		relayDifficultyMultiplierStr,
		targetDifficultyMultiplierStr,
	)

}
