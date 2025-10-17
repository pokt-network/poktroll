package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/pkg/polylog"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// NOTE: Please note that these messages are not of type `sdk.Msg`, and are therefore not a message/request
// that will be signable or invoke a state transition. However, following a similar `ValidateBasic` pattern
// allows us to localize & reuse validation logic.

// ValidateBasic performs basic (non-state-dependant) validation on a QueryGetSupplierRequest.
func (query *QueryGetSupplierRequest) ValidateBasic() error {
	// Validate the supplier operator address
	if _, err := sdk.AccAddressFromBech32(query.OperatorAddress); err != nil {
		return ErrSupplierInvalidAddress.Wrapf("invalid supplier operator address %s; (%v)", query.OperatorAddress, err)
	}

	return nil
}

// ValidateBasic performs basic (non-state-dependant) validation on a QueryAllSuppliersRequest.
func (query *QueryAllSuppliersRequest) ValidateBasic() error {
	logger := polylog.Ctx(context.TODO())

	// Track if any filter is set
	hasFilter := false

	// Validate service_id if provided
	if query.ServiceId != "" {
		hasFilter = true
		if err := sharedtypes.IsValidServiceId(query.ServiceId); err != nil {
			return ErrSupplierInvalidServiceId.Wrapf("%v", err.Error())
		}
	}

	// Validate operator_address if provided
	if query.OperatorAddress != "" {
		hasFilter = true
		if _, err := sdk.AccAddressFromBech32(query.OperatorAddress); err != nil {
			return ErrSupplierInvalidAddress.Wrapf("invalid operator address %s; (%v)", query.OperatorAddress, err)
		}
	}

	// Validate owner_address if provided
	if query.OwnerAddress != "" {
		hasFilter = true
		if _, err := sdk.AccAddressFromBech32(query.OwnerAddress); err != nil {
			return ErrSupplierInvalidAddress.Wrapf("invalid owner address %s; (%v)", query.OwnerAddress, err)
		}
	}

	if !hasFilter {
		logger.Info().Msg("No specific filter set when listing suppliers")
	}

	return nil
}
