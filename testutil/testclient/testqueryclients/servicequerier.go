package testqueryclients

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/pokt-network/poktroll/testutil/mockclient"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

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
