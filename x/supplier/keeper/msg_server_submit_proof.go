package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

func (k msgServer) SubmitProof(goCtx context.Context, msg *suppliertypes.MsgSubmitProof) (*suppliertypes.MsgSubmitProofResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.Logger(ctx).With("method", "SubmitProof")

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if _, err := k.ValidateSessionHeader(
		goCtx,
		msg.GetSessionHeader(),
		msg.GetSupplierAddress(),
	); err != nil {
		return nil, err
	}

	/*
			INCOMPLETE: Handling the message

		## Actions (error if anything fails)
		1. Retrieve a fully hydrated `session` from on-chain store using `msg` metadata
		2. Retrieve a fully hydrated `claim` from on-chain store using `msg` metadata
		3. Retrieve `relay.Req` and `relay.Res` from deserializing `proof.ClosestValueHash`

		## Basic Validations (metadata only)
		1. claim.sessionId == retrievedClaim.sessionId
		2. proof.sessionId == claim.sessionId
		3. msg.supplier in session.suppliers
		4. relay.Req.signer == session.appAddr
		5. relay.Res.signer == msg.supplier

		## Msg distribution validation (governance based params)
		1. Validate Proof submission is not too early; governance-based param + pseudo-random variation
		2. Validate Proof submission is not too late; governance-based param + pseudo-random variation

		## Relay Mining validation
		1. verify(proof.path) is the expected path; pseudo-random variation using on-chain data
		2. verify(proof.ValueHash, expectedDiffictul); governance based
		3. verify(claim.Root, proof.ClosestProof); verify the closest proof is correct
	*/

	//_ = ctx

	// Construct and insert proof after all validation.
	proof := suppliertypes.Proof{
		SupplierAddress:    msg.GetSupplierAddress(),
		SessionHeader:      msg.GetSessionHeader(),
		ClosestMerkleProof: msg.Proof,
	}
	k.Keeper.UpsertProof(ctx, proof)

	logger.
		With(
			"session_id", proof.GetSessionHeader().GetSessionId(),
			"session_end_height", proof.GetSessionHeader().GetSessionEndBlockHeight(),
			"supplier", proof.GetSupplierAddress(),
		).
		Debug("created proof")

	return &suppliertypes.MsgSubmitProofResponse{}, nil
}
