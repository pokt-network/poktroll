package sdk

import (
	"context"
	"net/http"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
)

// POKTRollSDK is the interface for the POKTRoll SDK. It is used by gateways
// and/or applications to interact with the Pocket protocol.
type POKTRollSDK interface {
	// GetApplications queries a list of all on-chain applications, optionally filtered by
	// delegated gateway
	GetApplications(ctx context.Context, options GetApplicationsOptions) ([]apptypes.Application, error)

	// GetSessionSupplierEndpoints returns the suppliers endpoints of the current session for
	// the given application and service.
	GetSessionSupplierEndpoints(
		ctx context.Context,
		appAddress string,
		serviceId string,
	) (session *SessionSuppliers, err error)

	// SendRelay sends a relay request to the given supplier's endpoint.
	SendRelay(
		ctx context.Context,
		sessionSupplierEndpoint *SingleSupplierEndpoint,
		request http.Request,
	) (response *servicetypes.RelayResponse, err error)
}
