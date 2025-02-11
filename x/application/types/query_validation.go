package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ValidateBasic performs basic (non-state-dependant) validation on a QueryAllApplicationsRequest.
func (query *QueryAllApplicationsRequest) ValidateBasic() error {
	gatewayDelegatedToAddr := query.GetGatewayAddressDelegatedTo()
	if gatewayDelegatedToAddr == "" {
		return nil
	}

	// Validate the delegation gateway address if the request specifies it as a constraint.
	if _, err := sdk.AccAddressFromBech32(gatewayDelegatedToAddr); err != nil {
		return ErrQueryAppsInvalidGatewayAddress.Wrapf("%q; (%v)", gatewayDelegatedToAddr, err)
	}

	return nil
}
