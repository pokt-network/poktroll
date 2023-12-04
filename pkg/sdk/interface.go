package sdk

import (
	"context"
	"net/http"

	servicetypes "github.com/pokt-network/poktroll/x/service/types"
)

// POKTRollSDK is the interface for the POKTRoll SDK. It is used by gateways
// to interact with the Pocket protocol.
type POKTRollSDK interface {

	// GetSession returns the suppliers endpoints of the current session for
	// the given application and service.
	GetCurrentSession(
		ctx context.Context,
		appAddress string,
		serviceId string,
	) (session []*SupplierEndpoint, err error)

	// SendRelay sends a relay request to the given supplier's endpoint.
	SendRelay(
		ctx context.Context,
		sessionSupplierEndpoint *SupplierEndpoint,
		request *http.Request,
	) (response *servicetypes.RelayResponse, err error)
}
