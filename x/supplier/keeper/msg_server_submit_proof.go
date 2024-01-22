package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

func (k msgServer) SubmitProof(goCtx context.Context, msg *suppliertypes.MsgSubmitProof) (*suppliertypes.MsgSubmitProofResponse, error) {
	// TODO_BLOCKER: Prevent Proof upserts after the tokenomics module has processes the respective session.
	// TODO_BLOCKER: Validate the signature on the Proof message corresponds to the supplier before Upserting.
	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.Logger(ctx).With("method", "SubmitProof")
	logger.Debug("submitting proof")

	/*
		TODO_INCOMPLETE: Handling the message

		## Actions (error if anything fails)
		1. Retrieve a fully hydrated `session` from on-chain store using `msg` metadata
		2. Retrieve a fully hydrated `claim` from on-chain store using `msg` metadata
		3. Retrieve `relay.Req` and `relay.Res` from deserializing `proof.ClosestValueHash`

		## Basic Validations (metadata only)
		1. proof.testSessionId == claim.testSessionId
		2. msg.supplier in session.suppliers
		3. relay.Req.signer == session.appAddr
		4. relay.Res.signer == msg.supplier

		## Msg distribution validation (governance based params)
		1. Validate Proof submission is not too early; governance-based param + pseudo-random variation
		2. Validate Proof submission is not too late; governance-based param + pseudo-random variation

		## Relay Mining validation
		1. verify(proof.path) is the expected path; pseudo-random variation using on-chain data
		2. verify(proof.ValueHash, expectedDiffictul); governance based
		3. verify(claim.Root, proof.ClosestProof); verify the closest proof is correct
	*/

	if err := msg.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if _, err := k.queryAndValidateSessionHeader(
		goCtx,
		msg.GetSessionHeader(),
		msg.GetSupplierAddress(),
	); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Construct and insert proof after all validation.
	proof := suppliertypes.Proof{
		SupplierAddress:    msg.GetSupplierAddress(),
		SessionHeader:      msg.GetSessionHeader(),
		ClosestMerkleProof: msg.Proof,
	}

	if err := k.queryAndValidateClaimForProof(ctx, &proof); err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	// TODO_BLOCKER: check if this proof already exists and return an appropriate error
	// in any case where the supplier should no longer be able to update the given proof.
	k.Keeper.UpsertProof(ctx, proof)

	// TODO_BLOCKER(@bryanchriswhite, @Olshansk): Call `tokenomics.SettleSessionAccounting()` here

	logger.
		With(
			"session_id", proof.GetSessionHeader().GetSessionId(),
			"session_end_height", proof.GetSessionHeader().GetSessionEndBlockHeight(),
			"supplier", proof.GetSupplierAddress(),
		).
		Debug("created proof")

	return &suppliertypes.MsgSubmitProofResponse{}, nil
}

// queryAndValidateClaimForProof ensures that  a claim corresponding to the given proof's
// session exists & has a matching supplier address and session header.
func (k msgServer) queryAndValidateClaimForProof(sdkCtx sdk.Context, proof *suppliertypes.Proof) error {
	sessionId := proof.GetSessionHeader().GetSessionId()
	// NB: no need to assert the testSessionId or supplier address as it is retrieved
	// by respective values of the give proof. I.e., if the claim exists, then these
	// values are guaranteed to match.
	claim, found := k.GetClaim(sdkCtx, sessionId, proof.GetSupplierAddress())
	if !found {
		return suppliertypes.ErrSupplierClaimNotFound.Wrapf("no claim found for session ID %q and supplier %q", sessionId, proof.GetSupplierAddress())
	}

	// Ensure session start heights match.
	if claim.GetSessionHeader().GetSessionStartBlockHeight() != proof.GetSessionHeader().GetSessionStartBlockHeight() {
		return suppliertypes.ErrSupplierInvalidSessionStartHeight.Wrapf(
			"claim session start height %d does not match proof session start height %d",
			claim.GetSessionHeader().GetSessionStartBlockHeight(),
			proof.GetSessionHeader().GetSessionStartBlockHeight(),
		)
	}

	// Ensure session end heights match.
	if claim.GetSessionHeader().GetSessionEndBlockHeight() != proof.GetSessionHeader().GetSessionEndBlockHeight() {
		return suppliertypes.ErrSupplierInvalidSessionEndHeight.Wrapf(
			"claim session end height %d does not match proof session end height %d",
			claim.GetSessionHeader().GetSessionEndBlockHeight(),
			proof.GetSessionHeader().GetSessionEndBlockHeight(),
		)
	}

	// Ensure application addresses match.
	if claim.GetSessionHeader().GetApplicationAddress() != proof.GetSessionHeader().GetApplicationAddress() {
		return suppliertypes.ErrSupplierInvalidAddress.Wrapf(
			"claim application address %q does not match proof application address %q",
			claim.GetSessionHeader().GetApplicationAddress(),
			proof.GetSessionHeader().GetApplicationAddress(),
		)
	}

	// Ensure service IDs match.
	if claim.GetSessionHeader().GetService().GetId() != proof.GetSessionHeader().GetService().GetId() {
		return suppliertypes.ErrSupplierInvalidService.Wrapf(
			"claim service ID %q does not match proof service ID %q",
			claim.GetSessionHeader().GetService().GetId(),
			proof.GetSessionHeader().GetService().GetId(),
		)
	}

	return nil
}
