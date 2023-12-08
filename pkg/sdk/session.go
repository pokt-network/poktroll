package sdk

import (
	"context"
	"net/url"

	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// SessionSuppliers is the structure that represents a session's end block height
// and its matching suppliers.
type SessionSuppliers struct {
	// Session is the fully hydrated session object returned by the query.
	Session *sessiontypes.Session

	// SuppliersEndpoints is a slice of the session's suppliers endpoints each
	// item representing a single supplier endpoint augmented with the session
	// header and the supplier's address.
	// An item from this slice is what needs to be passed to the `SendRelay`
	// function so it has all the information needed to send the relay request.
	SuppliersEndpoints []*SingleSupplierEndpoint
}

// SingleSupplierEndpoint is the structure that represents a supplier's endpoint
// augmented with the session's header and the supplier's address for easy
// access to the needed information when sending a relay request.
type SingleSupplierEndpoint struct {
	Url             *url.URL
	RpcType         sharedtypes.RPCType
	SupplierAddress string
	Header          *sessiontypes.SessionHeader
}

// GetSessionSupplierEndpoints returns a flattened structure of the endpoints
// from all suppliers in the session and returns them as a SupplierEndpoint slice.
// It queries for the latest session and caches it if the cached one is outdated.
func (sdk *poktrollSDK) GetSessionSupplierEndpoints(
	ctx context.Context,
	appAddress, serviceId string,
) (*SessionSuppliers, error) {
	sdk.serviceSessionSuppliersMu.RLock()
	defer sdk.serviceSessionSuppliersMu.RUnlock()

	latestBlockHeight := sdk.blockClient.LastNBlocks(ctx, 1)[0].Height()

	// Create the latestSessions map entry for the serviceId if it doesn't exist.
	if _, ok := sdk.serviceSessionSuppliers[serviceId]; !ok {
		sdk.serviceSessionSuppliers[serviceId] = map[string]*SessionSuppliers{}
	}

	// Create the latestSessions[serviceId] map entry for the appAddress if it doesn't exist.
	if _, ok := sdk.serviceSessionSuppliers[serviceId][appAddress]; !ok {
		sdk.serviceSessionSuppliers[serviceId][appAddress] = &SessionSuppliers{}
	}

	// currentSession is guaranteed to exist after the checks above.
	currentSession := sdk.serviceSessionSuppliers[serviceId][appAddress]

	// Return the current session's SuppliersEndpoints if the session is still valid.
	if currentSession.Session != nil &&
		latestBlockHeight < currentSession.Session.Header.SessionEndBlockHeight {
		return currentSession, nil
	}

	// Query for the current session.
	session, err := sdk.sessionQuerier.GetSession(
		ctx,
		appAddress,
		serviceId,
		latestBlockHeight,
	)
	if err != nil {
		return nil, err
	}

	// Override the old Session and SessionSuppliers and construct the new one.
	currentSession.Session = session
	currentSession.SuppliersEndpoints = []*SingleSupplierEndpoint{}

	for _, supplier := range session.Suppliers {
		for _, service := range supplier.Services {
			// Skip the session's services that don't match the requested serviceId.
			if service.Service.Id != serviceId {
				continue
			}

			// Loop through the services' endpoints and add them to the
			// SessionSuppliers.SuppliersEndpoints slice.
			for _, endpoint := range service.Endpoints {
				url, err := url.Parse(endpoint.Url)
				if err != nil {
					sdk.logger.Error().
						Str("url", endpoint.Url).
						Err(err).
						Msg("failed to parse url")
					continue
				}

				currentSession.SuppliersEndpoints = append(
					currentSession.SuppliersEndpoints,
					&SingleSupplierEndpoint{
						Url:             url,
						RpcType:         endpoint.RpcType,
						SupplierAddress: supplier.Address,
						Header:          session.Header,
					},
				)
			}
		}
	}

	return currentSession, nil
}
