package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

func (k msgServer) ValidateSessionHeader(
	goCtx context.Context,
	sessionHeader *sessiontypes.SessionHeader,
	supplierAddr string,
) (*sessiontypes.QueryGetSessionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.Logger(ctx).With("method", "SubmitProof")

	sessionReq := &sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: sessionHeader.GetApplicationAddress(),
		Service:            sessionHeader.GetService(),
		BlockHeight:        sessionHeader.GetSessionStartBlockHeight(),
	}

	sessionRes, err := k.Keeper.sessionKeeper.GetSession(goCtx, sessionReq)
	if err != nil {
		return nil, err
	}

	logger.
		With(
			"session_id", sessionRes.GetSession().GetSessionId(),
			"session_end_height", sessionHeader.GetSessionEndBlockHeight(),
			"supplier", supplierAddr,
		).
		Debug("got sessionId for proof")
	if sessionRes.Session.SessionId != sessionHeader.SessionId {
		return nil, suppliertypes.ErrSupplierInvalidSessionId.Wrapf(
			"claimed sessionRes ID does not match on-chain sessionRes ID; expected %q, got %q",
			sessionRes.Session.SessionId,
			sessionHeader.SessionId,
		)
	}
	return sessionRes, nil
}
