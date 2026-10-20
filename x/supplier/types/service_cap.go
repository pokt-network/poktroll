package types

import (
	"cmp"
	"math"
	"slices"

	sdkmath "cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// ServiceCapEnabled reports whether stake limits the number of services.
// A nil or zero stake_per_additional_service disables the cap.
func (p Params) ServiceCapEnabled() bool {
	return p.StakePerAdditionalService != nil && p.StakePerAdditionalService.IsPositive()
}

// minStakeOrDefault mirrors the nil fallback used by the unbonding paths.
func (p Params) minStakeOrDefault() cosmostypes.Coin {
	if p.MinStake == nil {
		return DefaultMinStake
	}
	return *p.MinStake
}

// minStakeIncludedServicesOrDefault returns min_stake_included_services, or its
// default (1) if it is zero. Validation never lets it be zero; the fallback keeps
// a zero in state from allowing no services at all.
func (p Params) minStakeIncludedServicesOrDefault() uint64 {
	if p.MinStakeIncludedServices == 0 {
		return DefaultMinStakeIncludedServices
	}
	return p.MinStakeIncludedServices
}

// AllowedServices returns how many services the given stake covers.
// limited is false when the cap is disabled, in which case allowed is meaningless.
//
// A stake below min_stake covers min_stake_included_services, the same as
// min_stake itself. Such a stake is rejected by staking and unbonded by
// settlement; when min_stake is raised it is left alone, and the cap must not
// treat it better (all services kept) or worse than a stake at the new minimum.
func AllowedServices(stake cosmostypes.Coin, params Params) (allowed uint64, limited bool) {
	if !params.ServiceCapEnabled() {
		return 0, false
	}

	minStake := params.minStakeOrDefault()
	included := params.minStakeIncludedServicesOrDefault()
	if stake.Amount.LT(minStake.Amount) {
		return included, true
	}

	additional := stake.Amount.Sub(minStake.Amount).Quo(params.StakePerAdditionalService.Amount)
	if !additional.IsUint64() || additional.Uint64() > math.MaxUint64-included {
		return math.MaxUint64, true
	}

	return included + additional.Uint64(), true
}

// RequiredStake returns the stake needed to serve numServices services.
// With the cap disabled it is min_stake.
func RequiredStake(numServices uint64, params Params) cosmostypes.Coin {
	minStake := params.minStakeOrDefault()
	included := params.minStakeIncludedServicesOrDefault()
	if !params.ServiceCapEnabled() || numServices <= included {
		return minStake
	}

	additional := sdkmath.NewIntFromUint64(numServices - included)
	return cosmostypes.NewCoin(minStake.Denom, minStake.Amount.Add(additional.Mul(params.StakePerAdditionalService.Amount)))
}

// IsCountedServiceConfig reports whether a config counts against the cap: it is
// active, or pending activation, and not deactivated by the next session start.
func IsCountedServiceConfig(config *sharedtypes.ServiceConfigUpdate, nextSessionStartHeight int64) bool {
	return config.DeactivationHeight == sharedtypes.NoDeactivationHeight ||
		config.DeactivationHeight > nextSessionStartHeight
}

// RankServiceConfigs returns configs ordered by the supplier's service priority.
// Services missing from priority (suppliers staked before the field existed) rank
// after listed ones, by service ID. The input slice is not modified; the returned
// slice holds the same pointers.
func RankServiceConfigs(
	configs []*sharedtypes.ServiceConfigUpdate,
	priority []string,
) []*sharedtypes.ServiceConfigUpdate {
	// Lookup only: map iteration order never reaches the output.
	rank := make(map[string]int, len(priority))
	for i, serviceId := range priority {
		if _, seen := rank[serviceId]; !seen {
			rank[serviceId] = i
		}
	}

	ranked := slices.Clone(configs)
	slices.SortStableFunc(ranked, func(a, b *sharedtypes.ServiceConfigUpdate) int {
		rankA, listedA := rank[a.Service.ServiceId]
		rankB, listedB := rank[b.Service.ServiceId]
		switch {
		case listedA && listedB:
			return cmp.Compare(rankA, rankB)
		case listedA:
			return -1
		case listedB:
			return 1
		default:
			return cmp.Compare(a.Service.ServiceId, b.Service.ServiceId)
		}
	})

	return ranked
}

// ServicePriorityFromConfigs returns the service IDs in message order.
func ServicePriorityFromConfigs(services []*sharedtypes.SupplierServiceConfig) []string {
	priority := make([]string, 0, len(services))
	for _, service := range services {
		priority = append(priority, service.ServiceId)
	}
	return priority
}
