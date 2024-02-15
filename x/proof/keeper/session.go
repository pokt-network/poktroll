package keeper

import (
	"context"

	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// queryAndValidateSessionHeader ensures that a session with the sessionID of the given session
// header exists and that this session includes the supplier with the given address.
func (k msgServer) queryAndValidateSessionHeader(
	ctx context.Context,
	sessionHeader *sessiontypes.SessionHeader,
	supplierAddr string,
) (*sessiontypes.Session, error) {
	logger := k.Logger().With("method", "SubmitProof")

	sessionReq := &sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: sessionHeader.GetApplicationAddress(),
		Service:            sessionHeader.GetService(),
		BlockHeight:        sessionHeader.GetSessionStartBlockHeight(),
	}

	// Get the on-chain session for the ground-truth against which the given
	// session header is to be validated.
	sessionRes, err := k.Keeper.sessionKeeper.GetSession(ctx, sessionReq)
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
			"session ID does not match on-chain session ID; expected %q, got %q",
			onChainSession.GetSessionId(),
			sessionHeader.GetSessionId(),
		)
	}

	// NB: it is redundant to assert that the service ID in the request matches the
	// on-chain session service ID because the session is queried using the service
	// ID as a parameter. Either a different session (i.e. different session ID)
	// or an error would be returned depending on whether an application/supplier
	// pair exists for the given service ID or not, respectively.

	// Ensure the given supplier is in the onChainSession supplier list.
	if found := foundSupplier(
		sessionRes.GetSession().GetSuppliers(),
		supplierAddr,
	); !found {
		return nil, suppliertypes.ErrSupplierNotFound.Wrapf(
			"supplier address %q not found in session ID %q",
			supplierAddr,
			sessionHeader.GetSessionId(),
		)
	}

	return onChainSession, nil
}

// foundSupplier ensures that the given supplier address is in the given list of suppliers.
func foundSupplier(suppliers []*sharedtypes.Supplier, supplierAddr string) bool {
	for _, supplier := range suppliers {
		if supplier.Address == supplierAddr {
			return true
		}
	}
	return false
}
