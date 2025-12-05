package relayer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
)

func TestNewRelayValidator(t *testing.T) {
	logger := polyzero.NewLogger()
	config := &ValidatorConfig{
		AllowedSupplierAddresses: []string{"pokt1supplier123", "pokt1supplier456"},
		GracePeriodExtraBlocks:   2,
	}

	validator := NewRelayValidator(logger, config, nil, nil, nil)
	require.NotNil(t, validator)

	rv, ok := validator.(*relayValidator)
	require.True(t, ok)
	require.Len(t, rv.allowedSuppliers, 2)
	require.Contains(t, rv.allowedSuppliers, "pokt1supplier123")
	require.Contains(t, rv.allowedSuppliers, "pokt1supplier456")
}

func TestRelayValidator_SetGetBlockHeight(t *testing.T) {
	logger := polyzero.NewLogger()
	config := &ValidatorConfig{}

	validator := NewRelayValidator(logger, config, nil, nil, nil)

	// Initial height should be 0
	require.Equal(t, int64(0), validator.GetCurrentBlockHeight())

	// Set height
	validator.SetCurrentBlockHeight(100)
	require.Equal(t, int64(100), validator.GetCurrentBlockHeight())

	// Update height
	validator.SetCurrentBlockHeight(200)
	require.Equal(t, int64(200), validator.GetCurrentBlockHeight())
}

func TestRelayValidator_BlockHeightConcurrency(t *testing.T) {
	logger := polyzero.NewLogger()
	config := &ValidatorConfig{}

	validator := NewRelayValidator(logger, config, nil, nil, nil)

	// Test concurrent access
	done := make(chan struct{})

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			validator.SetCurrentBlockHeight(int64(i))
		}
		done <- struct{}{}
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			_ = validator.GetCurrentBlockHeight()
		}
		done <- struct{}{}
	}()

	// Wait for both goroutines
	<-done
	<-done
}

func TestValidatorConfig_Empty(t *testing.T) {
	config := &ValidatorConfig{}

	require.Empty(t, config.AllowedSupplierAddresses)
	require.Equal(t, int64(0), config.GracePeriodExtraBlocks)
}

func TestValidatorConfig_WithValues(t *testing.T) {
	config := &ValidatorConfig{
		AllowedSupplierAddresses: []string{"supplier1", "supplier2"},
		GracePeriodExtraBlocks:   5,
	}

	require.Len(t, config.AllowedSupplierAddresses, 2)
	require.Equal(t, int64(5), config.GracePeriodExtraBlocks)
}

// MockRelayValidator is a mock implementation for testing dependent components.
type MockRelayValidator struct {
	validateFunc       func(context.Context, interface{}) error
	rewardEligibleFunc func(context.Context, interface{}) error
	blockHeight        int64
}

func (m *MockRelayValidator) ValidateRelayRequest(ctx context.Context, req interface{}) error {
	if m.validateFunc != nil {
		return m.validateFunc(ctx, req)
	}
	return nil
}

func (m *MockRelayValidator) CheckRewardEligibility(ctx context.Context, req interface{}) error {
	if m.rewardEligibleFunc != nil {
		return m.rewardEligibleFunc(ctx, req)
	}
	return nil
}

func (m *MockRelayValidator) GetCurrentBlockHeight() int64 {
	return m.blockHeight
}

func (m *MockRelayValidator) SetCurrentBlockHeight(height int64) {
	m.blockHeight = height
}

func TestMockRelayValidator(t *testing.T) {
	mock := &MockRelayValidator{
		blockHeight: 100,
	}

	require.Equal(t, int64(100), mock.GetCurrentBlockHeight())

	mock.SetCurrentBlockHeight(200)
	require.Equal(t, int64(200), mock.GetCurrentBlockHeight())
}

// Test interface compliance
func TestRelayValidator_InterfaceCompliance(t *testing.T) {
	logger := polyzero.NewLogger()
	config := &ValidatorConfig{}

	_ = NewRelayValidator(logger, config, nil, nil, nil)
}

func TestRelayValidator_EmptyAllowedSuppliers(t *testing.T) {
	logger := polyzero.NewLogger()
	config := &ValidatorConfig{
		AllowedSupplierAddresses: []string{},
	}

	validator := NewRelayValidator(logger, config, nil, nil, nil)
	rv, ok := validator.(*relayValidator)
	require.True(t, ok)
	require.Empty(t, rv.allowedSuppliers)
}
