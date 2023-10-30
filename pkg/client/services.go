package client

import (
	"fmt"

	sharedtypes "pocket/x/shared/types"
)

// NewTestApplicationServiceConfig returns a slice of application service configs for testing.
func NewTestApplicationServiceConfig(prefix string, count int) []*sharedtypes.ApplicationServiceConfig {
	appSvcCfg := make([]*sharedtypes.ApplicationServiceConfig, count)
	for i, _ := range appSvcCfg {
		serviceId := fmt.Sprintf("%s%d", prefix, i)
		appSvcCfg[i] = &sharedtypes.ApplicationServiceConfig{
			ServiceId: &sharedtypes.ServiceId{Id: serviceId},
		}
	}
	return appSvcCfg
}
