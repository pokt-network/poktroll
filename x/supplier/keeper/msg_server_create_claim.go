package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

func (k msgServer) CreateClaim(goCtx context.Context, msg *suppliertypes.MsgCreateClaim) (*suppliertypes.MsgCreateClaimResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.Logger(ctx).With("method", "CreateClaim")

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	sessionReq := &sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: msg.GetSessionHeader().GetApplicationAddress(),
		Service:            msg.GetSessionHeader().GetService(),
		BlockHeight:        msg.GetSessionHeader().GetSessionStartBlockHeight(),
	}
	sessionRes, err := k.Keeper.sessionKeeper.GetSession(goCtx, sessionReq)
	if err != nil {
		return nil, err
	}

	logger.
		With(
			"session_id", sessionRes.GetSession().GetSessionId(),
			"session_end_height", msg.GetSessionHeader().GetSessionEndBlockHeight(),
			"supplier", msg.GetSupplierAddress(),
		).
		Debug("got sessionId for claim")

	if sessionRes.Session.SessionId != msg.SessionHeader.SessionId {
		return nil, suppliertypes.ErrSupplierInvalidSessionId.Wrapf(
			"claimed sessionRes ID does not match on-chain sessionRes ID; expected %q, got %q",
			sessionRes.Session.SessionId,
			msg.SessionHeader.SessionId,
		)
	}

	var found bool
	for _, supplier := range sessionRes.GetSession().GetSuppliers() {
		if supplier.Address == msg.GetSupplierAddress() {
			found = true
			break
		}
	}

	if !found {
		return nil, suppliertypes.ErrSupplierNotFound.Wrapf(
			"supplier address %q in session ID %q",
			msg.GetSupplierAddress(),
			sessionRes.GetSession().GetSessionId(),
		)
	}

	logger.
		With(
			"session_id", sessionRes.GetSession().GetSessionId(),
			"session_end_height", msg.GetSessionHeader().GetSessionEndBlockHeight(),
			"supplier", msg.GetSupplierAddress(),
		).
		Debug("validated claim")

	/*
		TODO_INCOMPLETE:

		### Msg distribution validation (depends on sessionRes validation)
		1. [ ] governance-based earliest block offset
		2. [ ] pseudo-randomize earliest block offset

		### Claim validation
		1. [x] sessionRes validation
		2. [ ] msg distribution validation
	*/

	// Construct and insert claim after all validation.
	claim := suppliertypes.Claim{
		SupplierAddress:       msg.GetSupplierAddress(),
		SessionId:             msg.GetSessionHeader().GetSessionId(),
		SessionEndBlockHeight: uint64(msg.GetSessionHeader().GetSessionEndBlockHeight()),
		RootHash:              msg.RootHash,
	}
	k.Keeper.InsertClaim(ctx, claim)

	logger.
		With(
			"session_id", claim.GetSessionId(),
			"session_end_height", claim.GetSessionEndBlockHeight(),
			"supplier", claim.GetSupplierAddress(),
		).
		Debug("created claim")

	// TODO: return the claim in the response.
	return &suppliertypes.MsgCreateClaimResponse{}, nil
}
