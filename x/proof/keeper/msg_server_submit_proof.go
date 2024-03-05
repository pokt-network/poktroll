package keeper

import (
	"bytes"
	"context"
	"crypto/sha256"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
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
	// TODO_TECHDEBT: relayDifficultyBits should be a governance-based parameter
	relayDifficultyBits = 0

	// sumSize is the size of the sum of the relay request and response
	// in bytes. This is used to extract the relay request and response
	// from the closest merkle proof.
	// TODO_TECHDEBT: Have a method on the smst to extract the value hash or export
	// sumSize to be used instead of current local value
	sumSize = 8
)

func (k msgServer) SubmitProof(ctx context.Context, msg *types.MsgSubmitProof) (*types.MsgSubmitProofResponse, error) {
	// TODO_BLOCKER: Prevent Proof upserts after the tokenomics module has processes the respective session.
	logger := k.Logger().With("method", "SubmitProof")
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

	if err := msg.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
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

	// Verify the relay request and response session headers match the proof session header.
	if err := validateRelaySessionHeaders(relay, msg.GetSessionHeader()); err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	// Verify the relay response's signature.
	supplierAddress := msg.GetSupplierAddress()
	if err := k.verifyRelayResponseSignature(ctx, relay.GetRes(), supplierAddress); err != nil {
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

// queryAndValidateClaimForProof ensures that  a claim corresponding to the given proof's
// session exists & has a matching supplier address and session header.
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

// validateRelaySessionHeaders ensures that the relay request and response session headers
// match the proof session header.
func validateRelaySessionHeaders(
	relay *servicetypes.Relay,
	msgSessHeader *sessiontypes.SessionHeader,
) error {
	reqSessHeader := relay.GetReq().GetMeta().GetSessionHeader()
	respSessHeader := relay.GetRes().GetMeta().GetSessionHeader()

	// Ensure the relay request and response application addresses match
	// the proof application address.

	if reqSessHeader.GetApplicationAddress() != msgSessHeader.GetApplicationAddress() {
		return types.ErrProofInvalidRelay.Wrapf(
			"relay request application address %s does not match proof application address %s",
			reqSessHeader.GetApplicationAddress(),
			msgSessHeader.GetApplicationAddress(),
		)
	}

	if respSessHeader.GetApplicationAddress() != msgSessHeader.GetApplicationAddress() {
		return types.ErrProofInvalidRelay.Wrapf(
			"relay response application address %s does not match proof application address %s",
			reqSessHeader.GetApplicationAddress(),
			msgSessHeader.GetApplicationAddress(),
		)
	}

	// Ensure the relay request and response service IDs match the proof service ID.

	if reqSessHeader.GetService().GetId() != msgSessHeader.GetService().GetId() {
		return types.ErrProofInvalidRelay.Wrapf(
			"relay request service ID %s does not match proof service ID %s",
			reqSessHeader.GetService().GetId(),
			msgSessHeader.GetService().GetId(),
		)
	}

	if respSessHeader.GetService().GetId() != msgSessHeader.GetService().GetId() {
		return types.ErrProofInvalidRelay.Wrapf(
			"relay response service ID %s does not match proof service ID %s",
			respSessHeader.GetService().GetId(),
			msgSessHeader.GetService().GetId(),
		)
	}

	// Ensure the relay request and response session start block heights
	// match the proof session start block height.

	if reqSessHeader.GetSessionStartBlockHeight() != msgSessHeader.GetSessionStartBlockHeight() {
		return types.ErrProofInvalidRelay.Wrapf(
			"relay request session start height %d does not match proof session start height %d",
			reqSessHeader.GetSessionStartBlockHeight(),
			msgSessHeader.GetSessionStartBlockHeight(),
		)
	}

	if respSessHeader.GetSessionStartBlockHeight() != msgSessHeader.GetSessionStartBlockHeight() {
		return types.ErrProofInvalidRelay.Wrapf(
			"relay response session start height %d does not match proof session start height %d",
			respSessHeader.GetSessionStartBlockHeight(),
			msgSessHeader.GetSessionStartBlockHeight(),
		)
	}

	// Ensure the relay request and response session end block heights
	// match the proof session end block height.

	if reqSessHeader.GetSessionEndBlockHeight() != msgSessHeader.GetSessionEndBlockHeight() {
		return types.ErrProofInvalidRelay.Wrapf(
			"relay request session end height %d does not match proof session end height %d",
			reqSessHeader.GetSessionEndBlockHeight(),
			msgSessHeader.GetSessionEndBlockHeight(),
		)
	}

	if respSessHeader.GetSessionEndBlockHeight() != msgSessHeader.GetSessionEndBlockHeight() {
		return types.ErrProofInvalidRelay.Wrapf(
			"relay response session end height %d does not match proof session end height %d",
			respSessHeader.GetSessionEndBlockHeight(),
			msgSessHeader.GetSessionEndBlockHeight(),
		)
	}

	// Ensure the relay request and response session IDs match the proof session ID.

	if reqSessHeader.GetSessionId() != msgSessHeader.GetSessionId() {
		return types.ErrProofInvalidRelay.Wrapf(
			"relay request session ID %s does not match proof session ID %s",
			reqSessHeader.GetSessionId(),
			msgSessHeader.GetSessionId(),
		)
	}

	if respSessHeader.GetSessionId() != msgSessHeader.GetSessionId() {
		return types.ErrProofInvalidRelay.Wrapf(
			"relay response session ID %s does not match proof session ID %s",
			respSessHeader.GetSessionId(),
			msgSessHeader.GetSessionId(),
		)
	}

	return nil
}

// verifyRelayResponseSignature verifies the signature on the relay response.
// TODO_TECHDEBT: Factor out the relay response signature verification into a shared
// function that can be used by both the proof and the SDK packages.
func (k msgServer) verifyRelayResponseSignature(
	ctx context.Context,
	relayResponse *servicetypes.RelayResponse,
	supplierAddress string,
) error {
	// Get the account from the auth module
	accAddr, err := cosmostypes.AccAddressFromBech32(supplierAddress)
	if err != nil {
		return err
	}

	supplierAccount := k.accountKeeper.GetAccount(ctx, accAddr)

	// Get the public key from the account
	pubKey := supplierAccount.GetPubKey()
	if pubKey == nil {
		return types.ErrProofInvalidRelayResponse.Wrapf(
			"no public key found for supplier address %s",
			supplierAddress,
		)
	}

	supplierSignature := relayResponse.Meta.SupplierSignature

	// Get the relay response signable bytes and hash them.
	responseSignableBz, err := relayResponse.GetSignableBytesHash()
	if err != nil {
		return err
	}

	// Verify the relay response's signature
	if valid := pubKey.VerifySignature(responseSignableBz[:], supplierSignature); !valid {
		return types.ErrProofInvalidRelayResponse.Wrap("invalid relay response signature")
	}

	return nil
}

// verifyClosestProof verifies the closest merkle proof against the expected root hash.
func verifyClosestProof(
	proof *smt.SparseMerkleClosestProof,
	expectedRootHash []byte,
) error {
	spec := smt.NoPrehashSpec(sha256.New(), true)

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
	realyHash := hasher.Sum(nil)

	difficultyBits, err := protocol.CountDifficultyBits(realyHash)
	if err != nil {
		return types.ErrProofInvalidRelay.Wrapf(
			"error counting difficulty bits: %s",
			err,
		)
	}

	if difficultyBits < relayDifficultyBits {
		return types.ErrProofInvalidRelay.Wrapf(
			"relay difficulty %d is less than the required difficulty %d",
			difficultyBits,
			relayDifficultyBits,
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
	blockHeight := sessionHeader.GetSessionEndBlockHeight() + sessionkeeper.GetSessionGracePeriodBlockCount()
	blockHash := k.sessionKeeper.GetBlockHash(ctx, blockHeight)

	if !bytes.Equal(proof.Path, blockHash) {
		return types.ErrProofInvalidProof.Wrapf(
			"proof path %x does not match block hash %x",
			proof.Path,
			blockHash,
		)
	}

	return nil
}
