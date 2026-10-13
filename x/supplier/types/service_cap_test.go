package types

import (
	"math"
	"testing"

	sdkmath "cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/pocket"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func upokt(amount int64) cosmostypes.Coin { return cosmostypes.NewInt64Coin(pocket.DenomuPOKT, amount) }

func capParams(minStake, price int64, included uint64) Params {
	minStakeCoin, priceCoin := upokt(minStake), upokt(price)
	return Params{MinStake: &minStakeCoin, StakePerAdditionalService: &priceCoin, MinStakeIncludedServices: included}
}

func TestAllowedServices(t *testing.T) {
	tests := []struct {
		desc        string
		stake       int64
		params      Params
		wantAllowed uint64
		wantLimited bool
	}{
		{"price 0 disables the cap", 10, capParams(10, 0, 1), 0, false},
		{"nil price disables the cap", 10, Params{MinStake: &cosmostypes.Coin{Denom: pocket.DenomuPOKT, Amount: sdkmath.NewInt(10)}, MinStakeIncludedServices: 1}, 0, false},
		{"below min stake allows none", 9, capParams(10, 5, 1), 0, true},
		{"exactly min stake allows included", 10, capParams(10, 5, 3), 3, true},
		{"one upokt short of the next step", 14, capParams(10, 5, 3), 3, true},
		{"exactly the next step", 15, capParams(10, 5, 3), 4, true},
		{"many steps", 60, capParams(10, 5, 1), 11, true},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			allowed, limited := AllowedServices(upokt(test.stake), test.params)
			require.Equal(t, test.wantLimited, limited)
			require.Equal(t, test.wantAllowed, allowed)
		})
	}
}

func TestAllowedServices_SaturatesInsteadOfOverflowing(t *testing.T) {
	huge := cosmostypes.Coin{Denom: pocket.DenomuPOKT, Amount: sdkmath.NewIntFromUint64(math.MaxUint64).MulRaw(4)}
	allowed, limited := AllowedServices(huge, capParams(1, 1, math.MaxUint64-1))
	require.True(t, limited)
	require.Equal(t, uint64(math.MaxUint64), allowed)
}

func TestRequiredStake(t *testing.T) {
	params := capParams(10, 5, 3)
	require.Equal(t, upokt(10), RequiredStake(0, params))
	require.Equal(t, upokt(10), RequiredStake(3, params))
	require.Equal(t, upokt(15), RequiredStake(4, params))
	require.Equal(t, upokt(30), RequiredStake(7, params))
	// Required stake for N services is always allowed to serve N.
	for n := uint64(1); n <= 20; n++ {
		allowed, _ := AllowedServices(RequiredStake(n, params), params)
		require.GreaterOrEqual(t, allowed, n)
	}
}

func cfg(serviceId string, deactivationHeight int64) *sharedtypes.ServiceConfigUpdate {
	return &sharedtypes.ServiceConfigUpdate{
		Service:            &sharedtypes.SupplierServiceConfig{ServiceId: serviceId},
		DeactivationHeight: deactivationHeight,
	}
}

func TestIsCountedServiceConfig(t *testing.T) {
	require.True(t, IsCountedServiceConfig(cfg("a", sharedtypes.NoDeactivationHeight), 100))
	require.True(t, IsCountedServiceConfig(cfg("a", 120), 100))
	require.False(t, IsCountedServiceConfig(cfg("a", 100), 100))
	require.False(t, IsCountedServiceConfig(cfg("a", 80), 100))
}

func TestRankServiceConfigs(t *testing.T) {
	configs := []*sharedtypes.ServiceConfigUpdate{cfg("eth", 0), cfg("base", 0), cfg("sol", 0), cfg("arb", 0)}

	ranked := RankServiceConfigs(configs, []string{"sol", "eth"})
	require.Equal(t, []string{"sol", "eth", "arb", "base"}, serviceIds(ranked))

	// Empty priority (supplier staked before the field existed) falls back to service ID order.
	ranked = RankServiceConfigs(configs, nil)
	require.Equal(t, []string{"arb", "base", "eth", "sol"}, serviceIds(ranked))

	// The input slice is not reordered, and returned pointers are the input pointers.
	require.Equal(t, []string{"eth", "base", "sol", "arb"}, serviceIds(configs))
	require.Same(t, configs[0], RankServiceConfigs(configs, []string{"eth"})[0])
}

func TestServicePriorityFromConfigs(t *testing.T) {
	services := []*sharedtypes.SupplierServiceConfig{{ServiceId: "sol"}, {ServiceId: "eth"}}
	require.Equal(t, []string{"sol", "eth"}, ServicePriorityFromConfigs(services))
	require.Empty(t, ServicePriorityFromConfigs(nil))
}

func serviceIds(configs []*sharedtypes.ServiceConfigUpdate) []string {
	ids := make([]string, 0, len(configs))
	for _, config := range configs {
		ids = append(ids, config.Service.ServiceId)
	}
	return ids
}
