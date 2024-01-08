package supplier

import (
	"testing"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/pkg/either"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

const (
	testSessionNumber          = 1
	testBlockHeight            = 1
	testBlocksPerSession       = 4
	testSessionId              = "mock_session_id"
	defaultSessionKeyDelimiter = "/"
)

// TODO_IN_THIS_COMMIT: update godoc comment
//
// SessionFixtures is a map of session fixtures where the key is the
// application address and the value is the session fixture.
type SessionFixtures struct {
	sessionKeyDelimiter string
	sessions            map[string]*sessiontypes.Session
}

// AppSupplierPair is a pairing of an application and a supplier address.
type AppSupplierPair struct {
	AppAddr      string
	SupplierAddr string
}

// TODO_IN_THIS_COMMIT: update godoc comment
//
// NewSessionFixturesWithPairings creates a map of session fixtures where the key
// is the application address and the value is the session fixture. App/supplier
// addresses are expected to be provided in alternating order (as pairs). The same
// app and/or supplier may be given more than once but only distinct pairs will
// be added to the session fixtures map.
func NewSessionFixturesWithPairings(
	t *testing.T,
	service *sharedtypes.Service,
	appSupplierPairs ...*AppSupplierPair,
) *SessionFixtures {
	t.Helper()

	// Initialize the session fixtures map.
	sessionFixtures := &SessionFixtures{
		sessionKeyDelimiter: defaultSessionKeyDelimiter,
		sessions:            make(map[string]*sessiontypes.Session),
	}

	// Iterate over the app and supplier address pairs (two indices at a time),
	// and create a session fixture for each app address.
	for _, appSupplierPair := range appSupplierPairs {
		sessionFixtures.AddSession(t, appSupplierPair, service)
	}

	return sessionFixtures
}

// TODO_IN_THIS_COMMIT: godoc comment
func (s *SessionFixtures) AddSession(
	t *testing.T,
	pair *AppSupplierPair,
	service *sharedtypes.Service,
) {
	t.Helper()

	t.Logf("adding session with pair: %+v and service ID %q", pair, service.GetId())

	application := NewApplication(t, pair.AppAddr, service)
	supplier := NewSupplier(t, pair.SupplierAddr, service)

	_, ok := s.sessions[s.GetMockSessionId(pair)]
	// TODO_CONSIDERATION: could retrieve the session and append the supplier as all
	// other metadata is either constant or a function of the inputs to this function.
	require.Falsef(t, ok, "session already exists for app/supplier pair: %v", pair)

	mockSessionId := s.GetMockSessionId(pair)
	s.sessions[mockSessionId] = &sessiontypes.Session{
		Header: &sessiontypes.SessionHeader{
			ApplicationAddress:      pair.AppAddr,
			Service:                 service,
			SessionStartBlockHeight: testBlockHeight,
			SessionId:               mockSessionId,
			SessionEndBlockHeight:   testBlockHeight + testBlocksPerSession,
		},
		SessionId:           mockSessionId,
		SessionNumber:       testSessionNumber,
		NumBlocksPerSession: testBlocksPerSession,
		Application:         application,
		Suppliers:           []*sharedtypes.Supplier{supplier},
	}

}

// TODO_IN_THIS_COMMIT: godoc comment
func (s *SessionFixtures) GetSession(
	t *testing.T,
	pair *AppSupplierPair,
) *sessiontypes.Session {
	t.Helper()

	return s.sessions[s.GetMockSessionId(pair)]
}

// TODO_IN_THIS_COMMIT: godoc comment... assumes only one app service per session
func (s *SessionFixtures) GetSessionByAppService(
	t testing.TB,
	appAddr string,
	serviceId string,
) either.Either[*sessiontypes.Session] {
	t.Helper()

	var (
		err          error
		found        = false
		foundSession = new(sessiontypes.Session)
	)
	for _, session := range s.sessions {
		//t.Logf("sessionKey: %s, service ID: %s", k, session.GetHeader().GetService().GetId())
		//t.Logf("(debug) application addr: %s", session.GetHeader().GetApplicationAddress())

		if session.GetHeader().GetApplicationAddress() == appAddr &&
			session.GetHeader().GetService().GetId() == serviceId {
			foundSession = session
			found = true
			break
		}
	}
	if !found {
		// TODO_IMPROVE/NB: This error is necessary for sufficient mock behavior
		// in the context of a mock session keeper. Assertions made against this
		// error **do NOT necessarily** reflect the behavior of a real session keeper.
		// Eliminating the need for this error to be here would simplify this method
		// signature and usage.
		err = status.Error(codes.NotFound, types.ErrSupplierInvalidSessionId.Error())
	}

	return either.NewEither(foundSession, err)
}

// TODO_IN_THIS_COMMIT: godoc comment
func (s *SessionFixtures) GetMockSessionId(pair *AppSupplierPair) string {
	return pair.AppAddr + s.sessionKeyDelimiter + pair.SupplierAddr
}

// newSuppliers configures a supplier for the services provided and nil endpoints.
func NewSupplier(t *testing.T, addr string, services ...*sharedtypes.Service) *sharedtypes.Supplier {
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

// NewApplication configures an application for the services provided.
func NewApplication(t *testing.T, addr string, services ...*sharedtypes.Service) *apptypes.Application {
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
