package testqueryclients

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/pokt-network/poktroll/testutil/mockclient"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// services is a map of: serviceId -> Service
var services map[string]sharedtypes.Service

func init() {
	services = make(map[string]sharedtypes.Service)
}

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
		) (*sharedtypes.Service, error) {
			service, ok := services[serviceId]
			if !ok {
				return nil, fmt.Errorf("error while trying to retrieve a service")
			}

			return &service, nil
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
