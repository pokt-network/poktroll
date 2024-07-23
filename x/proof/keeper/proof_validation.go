package keeper

/*
	TODO_MAINNET: Document these steps in the docs and link here.

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

import (
	"bytes"
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/smt"

	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/x/proof/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

// IsProofValid validates the proof submitted by the supplier is correct with
// respect to an on-chain claim.
//
// This function should be called during session settlement (i.e. EndBlocker)
// rather than during proof submission (i.e. SubmitProof) because:
//  1. RPC requests should be quick, lightweight and only do basic validation
//  2. Validators are the ones responsible for the heavy processing & validation during state transitions
//  3. This creates an opportunity to slash suppliers who submit false proofs, whereas
//     they can keep retrying if it takes place in the SubmitProof handler.
func (k Keeper) IsProofValid(
	ctx context.Context,
	proof *types.Proof,
) (valid bool, err error) {
	logger := k.Logger().With("method", "ValidateProof")

	// Retrieve the supplier's public key.
	supplierAddr := proof.SupplierAddress
	supplierAccAddr, err := sdk.AccAddressFromBech32(supplierAddr)
	if err != nil {
		return false, err
	}
	supplierAccount := k.accountKeeper.GetAccount(ctx, supplierAccAddr)
	fmt.Println("OLSH", supplierAccAddr, supplierAccount.GetPubKey())
	// require.NotNil(t, acc)

	supplierPubKey, err := k.accountQuerier.GetPubKeyFromAddress(ctx, supplierAddr)
	if err != nil {
		return false, err
	}
	fmt.Println("OLSH3", supplierPubKey)

	// Validate the session header.
	var onChainSession *sessiontypes.Session
	onChainSession, err = k.queryAndValidateSessionHeader(ctx, proof.SessionHeader, supplierAddr)
	if err != nil {
		return false, err
	}
	logger.Info("queried and validated the session header")

	// Re-hydrate message session header with the on-chain session header.
	// This corrects for discrepancies between unvalidated fields in the session
	// header which can be derived from known values (e.g. session end height).
	sessionHeader := onChainSession.GetHeader()

	// Validate proof message commit height is within the respective session's
	// proof submission window using the on-chain session header.
	if err = k.validateProofWindow(ctx, sessionHeader, supplierAddr); err != nil {
		return false, err
	}

	if proof.ClosestMerkleProof == nil || len(proof.ClosestMerkleProof) == 0 {
		return false, types.ErrProofInvalidProof.Wrap("proof cannot be empty")
	}

	// Unmarshal the closest merkle proof from the message.
	sparseMerkleClosestProof := &smt.SparseMerkleClosestProof{}
	if err = sparseMerkleClosestProof.Unmarshal(proof.ClosestMerkleProof); err != nil {
		return false, types.ErrProofInvalidProof.Wrapf(
			"failed to unmarshal closest merkle proof: %s",
			err,
		)
	}

	// TODO_MAINNET(#427): Utilize smt.VerifyCompactClosestProof here to
	// reduce on-chain storage requirements for proofs.
	// Get the relay request and response from the proof.GetClosestMerkleProof.
	relayBz := sparseMerkleClosestProof.GetValueHash(&protocol.SmtSpec)
	relay := &servicetypes.Relay{}
	if err = k.cdc.Unmarshal(relayBz, relay); err != nil {
		return false, types.ErrProofInvalidRelay.Wrapf(
			"failed to unmarshal relay: %s",
			err,
		)
	}

	// Basic validation of the relay request.
	relayReq := relay.GetReq()
	if err = relayReq.ValidateBasic(); err != nil {
		return false, err
	}
	logger.Debug("successfully validated relay request")

	// Make sure that the supplier address in the proof matches the one in the relay request.
	if supplierAddr != relayReq.Meta.SupplierAddress {
		return false, types.ErrProofSupplierMismatch.Wrapf("supplier type mismatch")
	}
	logger.Debug("the proof supplier address matches the relay request supplier address")

	// Basic validation of the relay response.
	relayRes := relay.GetRes()
	if err = relayRes.ValidateBasic(); err != nil {
		return false, err
	}
	logger.Debug("successfully validated relay response")

	// Verify that the relay request session header matches the proof session header.
	if err = compareSessionHeaders(sessionHeader, relayReq.Meta.GetSessionHeader()); err != nil {
		return false, err
	}
	logger.Debug("successfully compared relay request session header")

	// Verify that the relay response session header matches the proof session header.
	if err = compareSessionHeaders(sessionHeader, relayRes.Meta.GetSessionHeader()); err != nil {
		return false, err
	}
	logger.Debug("successfully compared relay response session header")

	// Verify the relay request's signature.
	if err = k.ringClient.VerifyRelayRequestSignature(ctx, relayReq); err != nil {
		return false, err
	}
	logger.Debug("successfully verified relay request signature")

	// Verify the relay response's signature.
	if err = relayRes.VerifySupplierSignature(supplierPubKey); err != nil {
		return false, err
	}
	logger.Debug("successfully verified relay response signature")

	// Get the proof module's governance parameters.
	// TODO_FOLLOWUP(@olshansk, #690): Get the difficulty associated with the service
	params := k.GetParams(ctx)

	// Verify the relay difficulty is above the minimum required to earn rewards.
	if err = validateRelayDifficulty(
		relayBz,
		params.RelayDifficultyTargetHash,
		sessionHeader.Service.Id,
	); err != nil {
		return false, err
	}
	logger.Debug("successfully validated relay mining difficulty")

	// Validate that path the proof is submitted for matches the expected one
	// based on the pseudo-random on-chain data associated with the header.
	if err = k.validateClosestPath(
		ctx,
		sparseMerkleClosestProof,
		sessionHeader,
		supplierAddr,
	); err != nil {
		return false, err
	}
	logger.Debug("successfully validated proof path")

	// Retrieve the corresponding claim for the proof submitted so it can be
	// used in the proof validation below.
	claim, err := k.queryAndValidateClaimForProof(ctx, sessionHeader, supplierAddr)
	if err != nil {
		return false, err
	}

	logger.Debug("successfully retrieved and validated claim")

	// Verify the proof's closest merkle proof.
	if err = verifyClosestProof(sparseMerkleClosestProof, claim.GetRootHash()); err != nil {
		return false, err
	}
	logger.Debug("successfully verified closest merkle proof")

	return true, nil
}

// validateClosestPath ensures that the proof's path matches the expected path.
// Since the proof path needs to be pseudo-randomly selected AFTER the session
// ends, the seed for this is the block hash at the height when the proof window
// opens.
func (k Keeper) validateClosestPath(
	ctx context.Context,
	proof *smt.SparseMerkleClosestProof,
	sessionHeader *sessiontypes.SessionHeader,
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
	if !bytes.Equal(proof.Path, expectedProofPath) {
		return types.ErrProofInvalidProof.Wrapf(
			"the path of the proof provided (%x) does not match one expected by the on-chain protocol (%x)",
			proof.Path,
			expectedProofPath,
		)
	}

	return nil
}

// queryAndValidateClaimForProof ensures that a claim corresponding to the given
// proof's session exists & has a matching supplier address and session header,
// it then returns the corresponding claim if the validation is successful.
func (k Keeper) queryAndValidateClaimForProof(
	ctx context.Context,
	sessionHeader *sessiontypes.SessionHeader,
	supplierAddr string,
) (*types.Claim, error) {
	sessionId := sessionHeader.SessionId
	// NB: no need to assert the testSessionId or supplier address as it is retrieved
	// by respective values of the given proof. I.e., if the claim exists, then these
	// values are guaranteed to match.
	foundClaim, found := k.GetClaim(ctx, sessionId, supplierAddr)
	if !found {
		return nil, types.ErrProofClaimNotFound.Wrapf(
			"no claim found for session ID %q and supplier %q",
			sessionId,
			supplierAddr,
		)
	}

	claimSessionHeader := foundClaim.GetSessionHeader()
	proofSessionHeader := sessionHeader

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
	if sessionHeader.GetService().GetId() != expectedSessionHeader.GetService().GetId() {
		return types.ErrProofInvalidRelay.Wrapf(
			"session headers service IDs mismatch; expected: %q, got: %q",
			expectedSessionHeader.GetService().GetId(),
			sessionHeader.GetService().GetId(),
		)
	}

	// Compare the Service names.
	if sessionHeader.GetService().GetName() != expectedSessionHeader.GetService().GetName() {
		return types.ErrProofInvalidRelay.Wrapf(
			"sessionHeaders service names mismatch expect: %q, got: %q",
			expectedSessionHeader.GetService().GetName(),
			sessionHeader.GetService().GetName(),
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

// verifyClosestProof verifies the the correctness of the ClosestMerkleProof
// against the root hash committed to when creating the claim.
func verifyClosestProof(
	proof *smt.SparseMerkleClosestProof,
	claimRootHash []byte,
) error {
	valid, err := smt.VerifyClosestProof(proof, claimRootHash, &protocol.SmtSpec)
	if err != nil {
		return err
	}

	if !valid {
		return types.ErrProofInvalidProof.Wrap("invalid closest merkle proof")
	}

	return nil
}

// validateRelayDifficulty ensures that the relay's mining difficulty meets the
// required minimum threshold.
// TODO_TECHDEBT: Factor out the relay mining difficulty validation into a shared
// function that can be used by both the proof and the miner packages.
func validateRelayDifficulty(relayBz, targetHash []byte, serviceId string) error {
	relayHashArr := protocol.GetRelayHashFromBytes(relayBz)
	relayHash := relayHashArr[:]

	if len(targetHash) != protocol.RelayHasherSize {
		return types.ErrProofInvalidRelay.Wrapf(
			"invalid RelayDifficultyTargetHash: (%x); length wanted: %d; got: %d",
			targetHash,
			protocol.RelayHasherSize,
			len(targetHash),
		)
	}

	if !protocol.IsRelayVolumeApplicable(relayHash, targetHash) {
		var targetHashArr [protocol.RelayHasherSize]byte
		copy(targetHashArr[:], targetHash)

		relayDifficulty := protocol.GetDifficultyFromHash(relayHashArr)
		targetDifficulty := protocol.GetDifficultyFromHash(targetHashArr)

		return types.ErrProofInvalidRelay.Wrapf(
			"the difficulty relay being proven is (%d), and is smaller than the target difficulty (%d) for service %s",
			relayDifficulty,
			targetDifficulty,
			serviceId,
		)
	}

	return nil
}
