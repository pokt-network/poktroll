package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// queryAndValidateSessionHeader ensures that a session with the sessionID of the given session
// header exists and that this session includes the supplier with the given address.
func (k msgServer) queryAndValidateSessionHeader(
	goCtx context.Context,
	sessionHeader *sessiontypes.SessionHeader,
	supplierAddr string,
) (*sessiontypes.Session, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.Logger(ctx).With("method", "SubmitProof")

	sessionReq := &sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: sessionHeader.GetApplicationAddress(),
		Service:            sessionHeader.GetService(),
		BlockHeight:        sessionHeader.GetSessionStartBlockHeight(),
	}

	// Get the on-chain session for the ground-truth against which the given
	// session header is to be validated.
	sessionRes, err := k.Keeper.sessionKeeper.GetSession(goCtx, sessionReq)
	if err != nil {
		return nil, err
	}
	onChainSession := sessionRes.GetSession()

	logger.
		With(
			"session_id", onChainSession.GetSessionId(),
			"session_end_height", sessionHeader.GetSessionEndBlockHeight(),
			"supplier", supplierAddr,
		).
		Debug("got sessionId for proof")

	// Ensure that the given session header's session ID matches the on-chain onChainSession ID.
	if sessionHeader.GetSessionId() != onChainSession.GetSessionId() {
		return nil, suppliertypes.ErrSupplierInvalidSessionId.Wrapf(
			"claimed onChainSession ID does not match on-chain onChainSession ID; expected %q, got %q",
			onChainSession.GetSessionId(),
			sessionHeader.GetSessionId(),
		)
	}

	// Ensure the given supplier is in the onChainSession supplier list.
	var found bool
	for _, supplier := range sessionRes.GetSession().GetSuppliers() {
		if supplier.Address == supplierAddr {
			found = true
			break
		}
	}
	if !found {
		return nil, suppliertypes.ErrSupplierNotFound.Wrapf(
			"supplier address %q in onChainSession ID %q",
			supplierAddr,
			onChainSession.GetSessionId(),
		)
	}
	return onChainSession, nil
}
