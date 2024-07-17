package testqueryclients

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	sharedtypes "github.com/pokt-network/poktroll/proto/types/shared"
	"github.com/pokt-network/poktroll/testutil/mockclient"
)

// suppliersProvidedServicesMap is a map of maps:
//
//	supplierAddress -> {service -> []SupplierEndpoint}
//
// If an address is not present in the map it is then assumed that the supplier does
// not exist (has not staked)
var suppliersProvidedServicesMap map[string]map[string][]*sharedtypes.SupplierEndpoint

func init() {
	suppliersProvidedServicesMap = make(map[string]map[string][]*sharedtypes.SupplierEndpoint)
}

// NewTestSupplierQueryClient creates a mock of the SupplierQueryClient
// which allows the caller to call GetSupplier any times and will return
// an application with the given address.
func NewTestSupplierQueryClient(
	t *testing.T,
) *mockclient.MockSupplierQueryClient {
	ctrl := gomock.NewController(t)

	supplierQuerier := mockclient.NewMockSupplierQueryClient(ctrl)
	supplierQuerier.EXPECT().GetSupplier(gomock.Any(), gomock.Any()).
		DoAndReturn(func(
			_ context.Context,
			address string,
		) (supplier sharedtypes.Supplier, err error) {
			supplierProvidedServices, ok := suppliersProvidedServicesMap[address]
			if !ok {
				return sharedtypes.Supplier{}, errors.New("address not found")
			}

			services := []*sharedtypes.SupplierServiceConfig{}

			for serviceId, providedService := range supplierProvidedServices {
				serviceConfig := &sharedtypes.SupplierServiceConfig{
					Service: &sharedtypes.Service{
						Id: serviceId,
					},
					Endpoints: []*sharedtypes.SupplierEndpoint{},
				}

				for _, endpointConfig := range providedService {
					endpoint := &sharedtypes.SupplierEndpoint{
						Url:     endpointConfig.Url,
						RpcType: endpointConfig.RpcType,
					}
					serviceConfig.Endpoints = append(serviceConfig.Endpoints, endpoint)
				}

				services = append(services, serviceConfig)
			}

			return sharedtypes.Supplier{
				Address:  address,
				Services: services,
			}, nil
		}).
		AnyTimes()

	return supplierQuerier
}

// AddSupplierWithServiceEndpoints adds the given address and ServiceEndpoints
// to the suppliersProvidedServicesMap to mock it "existing" on chain,
// it will also remove the address from the map when the test is cleaned up.
func AddSuppliersWithServiceEndpoints(
	t *testing.T,
	address, service string,
	endpoints []*sharedtypes.SupplierEndpoint,
) {
	t.Helper()
	require.NotEmpty(t, endpoints)

	supplier, ok := suppliersProvidedServicesMap[address]
	if !ok {
		supplier = make(map[string][]*sharedtypes.SupplierEndpoint)
	}

	serviceEndpoints, ok := supplier[service]
	if !ok {
		serviceEndpoints = []*sharedtypes.SupplierEndpoint{}
	}

	serviceEndpoints = append(serviceEndpoints, endpoints...)

	supplier[service] = serviceEndpoints
	suppliersProvidedServicesMap[address] = supplier

	t.Cleanup(func() {
		delete(addressAccountMap, address)
	})
}
