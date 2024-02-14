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
	// TODO_BLOCKER: A potential issue with doing proof validation inside `SubmitProof` is that we will not
	// be storing false proofs on-chain (e.g. for slashing purposes). This could be considered a feature (e.g. less state bloat
	// against sybil attacks) or a bug (i.e. no mechanisms for slashing suppliers who submit false proofs). Revisit
	// this prior to mainnet launch as to whether the business logic for settling sessions should be in EndBlocker or here.
	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.Logger(ctx).With("method", "SubmitProof")
	logger.Info("About to start submitting proof")

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

	// Decomposing a few variables for easier access
	sessionHeader := msg.GetSessionHeader()
	supplierAddr := msg.GetSupplierAddress()

	// Helpers for logging the same metadata throughout this function calls
	logger = logger.With(
		"session_id", sessionHeader.GetSessionId(),
		"session_end_height", sessionHeader.GetSessionEndBlockHeight(),
		"supplier", supplierAddr)

	logger.Info("validated the submitProof message ")

	if _, err := k.queryAndValidateSessionHeader(
		goCtx,
		sessionHeader,
		supplierAddr,
	); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	logger.Info("queried and validated the session header")

	// Construct and insert proof after all validation.
	proof := suppliertypes.Proof{
		SupplierAddress:    supplierAddr,
		SessionHeader:      sessionHeader,
		ClosestMerkleProof: msg.Proof,
	}

	claim, err := k.queryAndValidateClaimForProof(ctx, &proof)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	logger.Info("queried and validated the claim")

	// TODO_BLOCKER: check if this proof already exists and return an appropriate error
	// in any case where the supplier should no longer be able to update the given proof.
	k.UpsertProof(ctx, proof)

	logger.Info("upserted the proof")
	logger.Info(string(ctx.TxBytes()))

	// TODO_BLOCKER: Revisit (per the comment above) as to whether this should be in `EndBlocker` or here.
	if err := k.tokenomicsKeeper.SettleSessionAccounting(ctx, claim); err != nil {
		return nil, err
	}

	logger.Info("settled session accounting")

	return &suppliertypes.MsgSubmitProofResponse{}, nil
}

// queryAndValidateClaimForProof ensures that  a claim corresponding to the given proof's
// session exists & has a matching supplier address and session header.
func (k msgServer) queryAndValidateClaimForProof(sdkCtx sdk.Context, proof *suppliertypes.Proof) (*suppliertypes.Claim, error) {
	sessionId := proof.GetSessionHeader().GetSessionId()
	// NB: no need to assert the testSessionId or supplier address as it is retrieved
	// by respective values of the given proof. I.e., if the claim exists, then these
	// values are guaranteed to match.
	claim, found := k.GetClaim(sdkCtx, sessionId, proof.GetSupplierAddress())
	if !found {
		return nil, suppliertypes.ErrSupplierClaimNotFound.Wrapf("no claim found for session ID %q and supplier %q", sessionId, proof.GetSupplierAddress())
	}

	// Ensure session start heights match.
	if claim.GetSessionHeader().GetSessionStartBlockHeight() != proof.GetSessionHeader().GetSessionStartBlockHeight() {
		return nil, suppliertypes.ErrSupplierInvalidSessionStartHeight.Wrapf(
			"claim session start height %d does not match proof session start height %d",
			claim.GetSessionHeader().GetSessionStartBlockHeight(),
			proof.GetSessionHeader().GetSessionStartBlockHeight(),
		)
	}

	// Ensure session end heights match.
	if claim.GetSessionHeader().GetSessionEndBlockHeight() != proof.GetSessionHeader().GetSessionEndBlockHeight() {
		return nil, suppliertypes.ErrSupplierInvalidSessionEndHeight.Wrapf(
			"claim session end height %d does not match proof session end height %d",
			claim.GetSessionHeader().GetSessionEndBlockHeight(),
			proof.GetSessionHeader().GetSessionEndBlockHeight(),
		)
	}

	// Ensure application addresses match.
	if claim.GetSessionHeader().GetApplicationAddress() != proof.GetSessionHeader().GetApplicationAddress() {
		return nil, suppliertypes.ErrSupplierInvalidAddress.Wrapf(
			"claim application address %q does not match proof application address %q",
			claim.GetSessionHeader().GetApplicationAddress(),
			proof.GetSessionHeader().GetApplicationAddress(),
		)
	}

	// Ensure service IDs match.
	if claim.GetSessionHeader().GetService().GetId() != proof.GetSessionHeader().GetService().GetId() {
		return nil, suppliertypes.ErrSupplierInvalidService.Wrapf(
			"claim service ID %q does not match proof service ID %q",
			claim.GetSessionHeader().GetService().GetId(),
			proof.GetSessionHeader().GetService().GetId(),
		)
	}

	return &claim, nil
}
