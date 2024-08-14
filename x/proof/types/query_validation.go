package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/pkg/polylog"
)

// NOTE: Please note that these messages are not of type `sdk.Msg`, and are therefore not a message/request
// that will be signable or invoke a state transition. However, following a similar `ValidateBasic` pattern
// allows us to localize & reuse validation logic.

// ValidateBasic performs basic (non-state-dependant) validation on a QueryGetClaimRequest.
func (query *QueryGetClaimRequest) ValidateBasic() error {
	// Validate the supplier operator address
	if _, err := sdk.AccAddressFromBech32(query.SupplierOperatorAddress); err != nil {
		return ErrProofInvalidAddress.Wrapf("invalid supplier operator address for claim being retrieved %s; (%v)", query.SupplierOperatorAddress, err)
	}

	if query.SessionId == "" {
		return ErrProofInvalidSessionId.Wrap("invalid empty session ID for claim being retrieved")
	}

	return nil
}

// ValidateBasic performs basic (non-state-dependant) validation on a QueryAllClaimsRequest.
func (query *QueryAllClaimsRequest) ValidateBasic() error {
	switch filter := query.Filter.(type) {
	case *QueryAllClaimsRequest_SupplierOperatorAddress:
		if _, err := sdk.AccAddressFromBech32(filter.SupplierOperatorAddress); err != nil {
			return ErrProofInvalidAddress.Wrapf("invalid supplier operator address for claims being retrieved %s; (%v)", filter.SupplierOperatorAddress, err)
		}

	case *QueryAllClaimsRequest_SessionId:
		if filter.SessionId == "" {
			return ErrProofInvalidSessionId.Wrap("invalid empty session ID for claims being retrieved")
		}

	case *QueryAllClaimsRequest_SessionEndHeight:
		// No validation needed for session end height.
	}

	return nil
}

func (query *QueryGetProofRequest) ValidateBasic() error {
	// Validate the supplier operator address
	if _, err := sdk.AccAddressFromBech32(query.SupplierOperatorAddress); err != nil {
		return ErrProofInvalidAddress.Wrapf("invalid supplier operator address for proof being retrieved %s; (%v)", query.SupplierOperatorAddress, err)
	}

	if query.SessionId == "" {
		return ErrProofInvalidSessionId.Wrap("invalid empty session ID for proof being retrieved")
	}

	return nil
}

func (query *QueryAllProofsRequest) ValidateBasic() error {
	// TODO_TECHDEBT: update function signature to receive a context.
	logger := polylog.Ctx(context.TODO())

	switch filter := query.Filter.(type) {
	case *QueryAllProofsRequest_SupplierOperatorAddress:
		if _, err := sdk.AccAddressFromBech32(filter.SupplierOperatorAddress); err != nil {
			return ErrProofInvalidAddress.Wrapf("invalid supplier operator address for proofs being retrieved %s; (%v)", filter.SupplierOperatorAddress, err)
		}

	case *QueryAllProofsRequest_SessionId:
		if filter.SessionId == "" {
			return ErrProofInvalidSessionId.Wrap("invalid empty session ID for proofs being retrieved")
		}

	case *QueryAllProofsRequest_SessionEndHeight:
		// No validation needed for session end height.

	default:
		// No filter is set
		logger.Info().Msg("No specific filter set when requesting proofs")
	}

	return nil
}
