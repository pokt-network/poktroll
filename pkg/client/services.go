package client

import (
	"fmt"

	"github.com/pokt-network/poktroll/proto/types/shared"
)

// NewTestApplicationServiceConfig returns a slice of application service configs for testing.
func NewTestApplicationServiceConfig(prefix string, count int) []*shared.ApplicationServiceConfig {
	appSvcCfg := make([]*shared.ApplicationServiceConfig, count)
	for i := range appSvcCfg {
		serviceId := fmt.Sprintf("%s%d", prefix, i)
		appSvcCfg[i] = &shared.ApplicationServiceConfig{
			Service: &shared.Service{Id: serviceId},
		}
	}
	return appSvcCfg
}
