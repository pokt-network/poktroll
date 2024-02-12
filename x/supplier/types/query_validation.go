package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// NOTE: Please note that these messages are not of type `sdk.Msg`, and are therefore not a message/request
// that will be signable or invoke a state transition. However, following a similar `ValidateBasic` pattern
// allows us to localize & reuse validation logic.

// ValidateBasic performs basic (non-state-dependant) validation on a QueryGetClaimRequest.
func (query *QueryGetClaimRequest) ValidateBasic() error {
	// Validate the supplier address
	if _, err := sdk.AccAddressFromBech32(query.SupplierAddress); err != nil {
		return ErrSupplierInvalidAddress.Wrapf("invalid supplier address for claim being retrieved %s; (%v)", query.SupplierAddress, err)
	}

	// TODO_TECHDEBT: Validate the session ID once we have a deterministic way to generate it
	if query.SessionId == "" {
		return ErrSupplierInvalidSessionId.Wrapf("invalid session ID for claim being retrieved %s", query.SessionId)
	}

	return nil
}

// ValidateBasic performs basic (non-state-dependant) validation on a QueryAllClaimsRequest.
func (query *QueryAllClaimsRequest) ValidateBasic() error {
	// TODO_UPNEXT(@bryanchriswhite #378): uncommment once off-chain pkgs are available.
	//logger := polylog.Ctx(context.Background())

	switch filter := query.Filter.(type) {
	case *QueryAllClaimsRequest_SupplierAddress:
		if _, err := sdk.AccAddressFromBech32(filter.SupplierAddress); err != nil {
			return ErrSupplierInvalidAddress.Wrapf("invalid supplier address for claims being retrieved %s; (%v)", filter.SupplierAddress, err)
		}

	case *QueryAllClaimsRequest_SessionId:
		// TODO_UPNEXT(@bryanchriswhite #378): uncommment once off-chain pkgs are available.
		//logger.Warn().
		//	Str("session_id", filter.SessionId).
		//	Msg("TODO_TECHDEBT: Validate the session ID once we have a deterministic way to generate it")

	case *QueryAllClaimsRequest_SessionEndHeight:
		if filter.SessionEndHeight < 0 {
			return ErrSupplierInvalidSessionEndHeight.Wrapf("invalid session end height for claims being retrieved %d", filter.SessionEndHeight)
		}
	}

	return nil
}

func (query *QueryGetProofRequest) ValidateBasic() error {
	// Validate the supplier address
	if _, err := sdk.AccAddressFromBech32(query.SupplierAddress); err != nil {
		return ErrSupplierInvalidAddress.Wrapf("invalid supplier address for proof being retrieved %s; (%v)", query.SupplierAddress, err)
	}

	// TODO_TECHDEBT: Validate the session ID once we have a deterministic way to generate it
	if query.SessionId == "" {
		return ErrSupplierInvalidSessionId.Wrapf("invalid session ID for proof being retrieved %s", query.SessionId)
	}

	return nil
}

func (query *QueryAllProofsRequest) ValidateBasic() error {
	// TODO_UPNEXT(@bryanchriswhite #378): uncommment once off-chain pkgs are available.
	//logger := polylog.Ctx(context.TODO())

	switch filter := query.Filter.(type) {
	case *QueryAllProofsRequest_SupplierAddress:
		if _, err := sdk.AccAddressFromBech32(filter.SupplierAddress); err != nil {
			return ErrSupplierInvalidAddress.Wrapf("invalid supplier address for proofs being retrieved %s; (%v)", filter.SupplierAddress, err)
		}

	case *QueryAllProofsRequest_SessionId:
		// TODO_UPNEXT(@bryanchriswhite #378): uncommment once off-chain pkgs are available.
		//logger.Warn().
		//	Str("session_id", filter.SessionId).
		//	Msg("TODO_TECHDEBT: Validate the session ID once we have a deterministic way to generate it")

	case *QueryAllProofsRequest_SessionEndHeight:
		if filter.SessionEndHeight < 0 {
			return ErrSupplierInvalidSessionEndHeight.Wrapf("invalid session end height for proofs being retrieved %d", filter.SessionEndHeight)
		}

	default:
		// No filter is set
		// TODO_UPNEXT(@bryanchriswhite #378): uncommment once off-chain pkgs are available.
		//logger.Debug().Msg("No specific filter set when requesting proofs")
	}

	return nil
}
