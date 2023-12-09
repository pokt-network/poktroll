package client

import (
	"fmt"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// NewTestApplicationServiceConfig returns a slice of application service configs for testing.
func NewTestApplicationServiceConfig(prefix string, count int) []*sharedtypes.ApplicationServiceConfig {
	appSvcCfg := make([]*sharedtypes.ApplicationServiceConfig, count)
	for i := range appSvcCfg {
		serviceId := fmt.Sprintf("%s%d", prefix, i)
		appSvcCfg[i] = &sharedtypes.ApplicationServiceConfig{
			Service: &sharedtypes.Service{Id: serviceId},
		}
	}
	return appSvcCfg
}
