package testqueryclients

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/pokt-network/pocket/testutil/mockclient"
	prooftypes "github.com/pokt-network/pocket/x/proof/types"
	servicetypes "github.com/pokt-network/pocket/x/service/types"
	sharedtypes "github.com/pokt-network/pocket/x/shared/types"
)

// TODO_TECHDEBT: refactor the methods using this variable to avoid having a global scope
// for the map across unit tests run under the same testing.T instance.
// Ditto for other similar package-level variables in this package.
// relayDifficultyTargets is a map of: serviceId -> RelayMiningDifficulty
// It is updated by the SetServiceRelayDifficultyTargetHash, and read by
// the mock tokenomics query client to get a specific service's relay difficulty
// target hash.
var relayDifficultyTargets = make(map[string]*servicetypes.RelayMiningDifficulty)

// TODO_TECHDEBT: refactor the methods using this variable to avoid having a global scope
// for the map across unit tests run under the same testing.T instance.
// services is a map of: serviceId -> Service
var services = make(map[string]sharedtypes.Service)

// NewTestSessionQueryClient creates a mock of the SessionQueryClient
// which allows the caller to call GetSession any times and will return
// the session matching the app address, serviceID and the blockHeight passed.
func NewTestServiceQueryClient(
	t *testing.T,
) *mockclient.MockServiceQueryClient {
	t.Helper()
	ctrl := gomock.NewController(t)

	serviceQuerier := mockclient.NewMockServiceQueryClient(ctrl)
	serviceQuerier.EXPECT().GetService(gomock.Any(), gomock.Any()).
		DoAndReturn(func(
			_ context.Context,
			serviceId string,
		) (sharedtypes.Service, error) {
			service, ok := services[serviceId]
			if !ok {
				return sharedtypes.Service{}, prooftypes.ErrProofServiceNotFound.Wrapf("service %s not found", serviceId)
			}

			return service, nil
		}).
		AnyTimes()

	serviceQuerier.EXPECT().GetServiceRelayDifficulty(gomock.Any(), gomock.Any()).
		DoAndReturn(func(
			_ context.Context,
			serviceId string,
		) (servicetypes.RelayMiningDifficulty, error) {
			relayDifficulty, ok := relayDifficultyTargets[serviceId]
			if !ok {
				return servicetypes.RelayMiningDifficulty{}, servicetypes.ErrServiceMissingRelayMiningDifficulty.Wrapf("retrieving the relay mining difficulty for service %s", serviceId)
			}

			return *relayDifficulty, nil
		}).
		AnyTimes()

	return serviceQuerier
}

// AddToExistingServices adds the given service to the services map to mock it "existing"
// on chain, it will also remove the services from the map when the test is cleaned up.
func AddToExistingServices(
	t *testing.T,
	service sharedtypes.Service,
) {
	t.Helper()

	services[service.Id] = service

	t.Cleanup(func() {
		delete(services, service.Id)
	})
}

// AddServiceRelayDifficultyTargetHash sets the relay difficulty target hash
// for the given service to mock it "existing" on chain.
// It will also remove the service relay difficulty target hashes from the map when the test is cleaned up.
func SetServiceRelayDifficultyTargetHash(t *testing.T,
	serviceId string,
	relayDifficultyTargetHash []byte,
) {
	t.Helper()

	relayDifficultyTargets[serviceId] = &servicetypes.RelayMiningDifficulty{
		ServiceId:  serviceId,
		TargetHash: relayDifficultyTargetHash,
	}

	t.Cleanup(func() {
		delete(relayDifficultyTargets, serviceId)
	})
}
