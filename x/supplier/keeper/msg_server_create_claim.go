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

	sessionRes, err := k.Keeper.sessionKeeper.GetSession(goCtx, &sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: msg.SessionHeader.ApplicationAddress,
		Service:            msg.SessionHeader.Service,
		BlockHeight:        msg.SessionHeader.SessionStartBlockHeight,
	})
	if err != nil {
		return nil, err
	}

	if sessionRes.Session.SessionId != msg.SessionHeader.SessionId {
		return nil, suppliertypes.ErrSupplierInvalidSessionId.Wrapf(
			"claimed sessionRes ID does not match on-chain sessionRes ID; expected %q, got %q",
			sessionRes.Session.SessionId,
			msg.SessionHeader.SessionId,
		)
	}

	var found bool
	for _, supplier := range sessionRes.GetSession().GetSuppliers() {
		if supplier.Address == msg.SupplierAddress {
			found = true
			break
		}
	}

	if !found {
		return nil, suppliertypes.ErrSupplierNotFound.Wrapf(
			"supplier address %q in session ID %q",
			msg.SupplierAddress,
			sessionRes.GetSession().GetSessionId(),
		)
	}

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
		SupplierAddress:       msg.SupplierAddress,
		SessionId:             msg.SessionHeader.SessionId,
		SessionEndBlockHeight: uint64(msg.SessionHeader.SessionEndBlockHeight),
		RootHash:              msg.RootHash,
	}
	k.Keeper.InsertClaim(ctx, claim)

	logger.Info("created claim for supplier %s at sessionRes ending height %d", claim.SupplierAddress, claim.SessionEndBlockHeight)

	return &suppliertypes.MsgCreateClaimResponse{}, nil
}
