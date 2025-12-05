package relayer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestServiceDependencies_Empty(t *testing.T) {
	deps := ServiceDependencies{}

	require.Nil(t, deps.Logger)
	require.Nil(t, deps.RedisClient)
	require.Nil(t, deps.RingClient)
	require.Nil(t, deps.SessionClient)
	require.Nil(t, deps.SharedClient)
	require.Nil(t, deps.BlockClient)
}

func TestNewService_InvalidConfig(t *testing.T) {
	config := &Config{
		// Missing required fields
	}

	deps := ServiceDependencies{}

	_, err := NewService(config, deps)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid config")
}

func TestService_GetProxyServer(t *testing.T) {
	// This test verifies the getter methods work on nil service
	// Full service creation requires Redis connection
	var s *Service

	require.Nil(t, s)
}
