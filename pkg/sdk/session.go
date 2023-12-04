package sdk

import (
	"context"
	"log"
	"net/url"

	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// sessionSuppliers is the structure that represents a session's end block height
// and its matching suppliers.
type sessionSuppliers struct {
	// SessionEndBlockHeight is the session's end block height that helps
	// determine if the session is still valid without looking into SupplierEndpoints slice.
	SessionEndBlockHeight int64
	SuppliersEndpoints    []*SupplierEndpoint
}

// SupplierEndpoint is the structure that represents a supplier's endpoint
// augmented with the session's header and the supplier's address for easy
// access to the needed information when sending a relay request.
type SupplierEndpoint struct {
	Url             *url.URL
	RpcType         sharedtypes.RPCType
	SupplierAddress string
	Header          *sessiontypes.SessionHeader
}

// GetCurrentSession gets the current session for the given service
// It returns the current state if it exists and is still valid, otherwise it
// queries for the latest session, constructs the SessionSuppliers, caches and
// returns its SupplierEndpoints.
func (sdk *poktrollSDK) GetCurrentSession(
	ctx context.Context,
	appAddress, serviceId string,
) ([]*SupplierEndpoint, error) {
	sdk.sessionMu.RLock()
	defer sdk.sessionMu.RUnlock()

	latestBlockHeight := sdk.blockClient.LatestBlock(ctx).Height()

	// Create the latestSessions map entry for the serviceId if it doesn't exist.
	if _, ok := sdk.latestSessions[serviceId]; !ok {
		sdk.latestSessions[serviceId] = map[string]*sessionSuppliers{}
	}

	// Create the latestSessions[serviceId] map entry for the appAddress if it doesn't exist.
	if _, ok := sdk.latestSessions[serviceId][appAddress]; !ok {
		sdk.latestSessions[serviceId][appAddress] = &sessionSuppliers{}
	}

	currentSession := sdk.latestSessions[serviceId][appAddress]

	// Return the current session's SuppliersEndpoints if the session is still valid.
	if latestBlockHeight < currentSession.SessionEndBlockHeight {
		return currentSession.SuppliersEndpoints, nil
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

	// Override the old SessionSuppliers and constructs the new one.
	currentSession.SessionEndBlockHeight = session.Header.SessionEndBlockHeight
	currentSession.SuppliersEndpoints = []*SupplierEndpoint{}

	for _, supplier := range session.Suppliers {
		for _, services := range supplier.Services {
			// Skip the session's services that don't match the requested serviceId.
			if services.Service.Id != serviceId {
				continue
			}

			// Loop through the services' endpoints and add them to the
			// SessionSuppliers.SuppliersEndpoints slice.
			for _, endpoint := range services.Endpoints {
				url, err := url.Parse(endpoint.Url)
				if err != nil {
					log.Printf("error parsing url: %s", err)
					continue
				}

				currentSession.SuppliersEndpoints = append(
					currentSession.SuppliersEndpoints,
					&SupplierEndpoint{
						Url:             url,
						RpcType:         endpoint.RpcType,
						SupplierAddress: supplier.Address,
						Header:          session.Header,
					},
				)
			}
		}
	}

	return currentSession.SuppliersEndpoints, nil
}
