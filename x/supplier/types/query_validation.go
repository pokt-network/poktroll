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
	// TODO_TECHDEBT: update function signature to receive a context.
	logger := polylog.Ctx(context.TODO())

	switch filter := query.Filter.(type) {
	case *QueryAllSuppliersRequest_ServiceId:
		// If the service ID is set, check if it's valid
		if filter.ServiceId != "" && !sharedtypes.IsValidServiceId(filter.ServiceId) {
			return ErrSupplierInvalidServiceId.Wrap("invalid empty service ID for suppliers being retrieved")
		}

	default:
		// No filter is set
		logger.Info().Msg("No specific filter set when listing suppliers")
	}

	return nil
}
