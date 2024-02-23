package sdk

import (
	"context"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
)

// GetApplicationsOptions defines the options for the GetApplications SDK method
type GetApplicationsOptions struct {
	// If non-empty, only return applications that have delegated to this gateway address
	DelegatedGatewayAddress string
}

// GetApplications queries a list of all on-chain applications, filtered by the
// options provided
func (sdk *poktrollSDK) GetApplications(ctx context.Context, options GetApplicationsOptions) ([]apptypes.Application, error) {
	delegatingApplications := []apptypes.Application{}

	applications, err := sdk.applicationQuerier.GetAllApplications(ctx)
	if err != nil {
		return nil, ErrSDKGetApplications.Wrapf("error querying applications: %s", err)
	}

	if options.DelegatedGatewayAddress == "" {
		return applications, nil
	}

	// TODO_CONSIDERATION: Look into updating the on-chain `QueryAllApplicationsRequest` for filtering
	// options to avoid needing to do it on the client side.
	for _, app := range applications {
		for _, delegatedGatewayAddress := range app.DelegateeGatewayAddresses {
			if delegatedGatewayAddress == options.DelegatedGatewayAddress {
				delegatingApplications = append(delegatingApplications, app)
			}
		}
	}

	return delegatingApplications, nil
}
