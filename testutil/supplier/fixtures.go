package supplier

import (
	"testing"

	sdktypes "github.com/cosmos/cosmos-sdk/types"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const (
	testSessionNumber    = 1
	testBlockHeight      = 1
	testBlocksPerSession = 4
	testSessionId        = "mock_session_id"
)

// SessionsByAppAddress is a map of session fixtures where the key is the
// application address and the value is the session fixture.
type SessionsByAppAddress map[string]sessiontypes.Session

// NewSessionFixturesWithPairings creates a map of session fixtures where the key
// is the application address and the value is the session fixture. App/supplier
// addresses are expected to be provided in alternating order (as pairs). The same
// app and/or supplier may be given more than once but only distinct pairs will
// be added to the session fixtures map.
func NewSessionFixturesWithPairings(
	t *testing.T,
	service *sharedtypes.Service,
	appAndSupplierAddrPairs ...string,
) SessionsByAppAddress {
	t.Helper()

	if len(appAndSupplierAddrPairs)%2 != 0 {
		t.Fatalf("expected even number of app and supplier address pairs, got %d", len(appAndSupplierAddrPairs))
	}

	// Initialize the session fixtures map.
	sessionFixturesByAppAddr := make(SessionsByAppAddress)

	// Iterate over the app and supplier address pairs (two indices at a time),
	// and create a session fixture for each app address.
	for i := 0; i < len(appAndSupplierAddrPairs); i += 2 {
		appAddr := appAndSupplierAddrPairs[i]
		application := newApplication(t, appAddr, service)

		supplierAddr := appAndSupplierAddrPairs[i+1]
		supplier := newSupplier(t, supplierAddr, service)

		if session, ok := sessionFixturesByAppAddr[appAddr]; ok {
			session.Suppliers = append(session.Suppliers, supplier)
			continue
		}

		sessionFixturesByAppAddr[appAddr] = sessiontypes.Session{
			Header: &sessiontypes.SessionHeader{
				ApplicationAddress:      appAddr,
				Service:                 service,
				SessionStartBlockHeight: testBlockHeight,
				SessionId:               testSessionId,
				SessionEndBlockHeight:   testBlockHeight + testBlocksPerSession,
			},
			SessionId:           testSessionId,
			SessionNumber:       testSessionNumber,
			NumBlocksPerSession: testBlocksPerSession,
			Application:         application,
			Suppliers: []*sharedtypes.Supplier{
				newSupplier(t, supplierAddr, service),
			},
		}
	}

	return sessionFixturesByAppAddr
}

// newSuppliers configures a supplier for the services provided and nil endpoints.
func newSupplier(t *testing.T, addr string, services ...*sharedtypes.Service) *sharedtypes.Supplier {
	t.Helper()

	serviceConfigs := make([]*sharedtypes.SupplierServiceConfig, len(services))
	for i, service := range services {
		serviceConfigs[i] = &sharedtypes.SupplierServiceConfig{
			Service:   service,
			Endpoints: nil,
		}
	}

	return &sharedtypes.Supplier{
		Address:  addr,
		Stake:    &sdktypes.Coin{},
		Services: serviceConfigs,
	}
}

// newApplication configures an application for the services provided.
func newApplication(t *testing.T, addr string, services ...*sharedtypes.Service) *apptypes.Application {
	t.Helper()

	serviceConfigs := make([]*sharedtypes.ApplicationServiceConfig, len(services))
	for i, service := range services {
		serviceConfigs[i] = &sharedtypes.ApplicationServiceConfig{
			Service: service,
		}
	}

	return &apptypes.Application{
		Address:                   addr,
		Stake:                     &sdktypes.Coin{},
		ServiceConfigs:            serviceConfigs,
		DelegateeGatewayAddresses: nil,
	}
}
