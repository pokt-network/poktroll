package types

import (
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// ApplicationNotUnstaking is the value of `unstake_session_end_height` if the
// application is not actively in the unbonding period.
const ApplicationNotUnstaking uint64 = iota

// IsUnbonding returns true if the application is actively unbonding.
// It determines if the application has submitted an unstake message, in which case
// the application has its UnstakeSessionEndHeight set.
func (s *Application) IsUnbonding() bool {
	return s.UnstakeSessionEndHeight != ApplicationNotUnstaking
}

// HasPendingTransfer returns true if the application has begun but not completed
// an application transfer. It determines if the application has submitted a transfer
// message, in which case the application has its PendingTransfer field set.
func (s *Application) HasPendingTransfer() bool {
	return s.PendingTransfer != nil
}

// IsActive returns whether the application is allowed to request services at the
// given query height.
// An application that has not submitted an unstake message is always active.
// An application that has submitted an unstake message is active until the end of
// the session containing the height at which unstake message was submitted.
// An application that has a pending transfer is active until the end of the session
// containing the height at which the transfer was initiated.
func (s *Application) IsActive(queryHeight int64) bool {
	if s.IsUnbonding() {
		return uint64(queryHeight) <= s.GetUnstakeSessionEndHeight()
	}
	if s.HasPendingTransfer() {
		return uint64(queryHeight) <= s.GetPendingTransfer().GetSessionEndHeight()
	}
	return true
}

// IsActive checks if the ApplicationServiceConfigUpdate is active at the given
// block height.
//
// A service configuration is active when:
//  1. The query height is greater than or equal to the activation height, AND
//  2. Either the deactivation height is 0 (no scheduled deactivation), or the
//     query height is strictly less than the deactivation height.
//
// This mirrors the supplier sharedtypes.ServiceConfigUpdate.IsActive semantics
// so application and supplier history use identical activation windows.
func (s *ApplicationServiceConfigUpdate) IsActive(queryHeight int64) bool {
	// Activation height is in the future: not active yet.
	if s.ActivationHeight > queryHeight {
		return false
	}

	// No deactivation scheduled (0): active indefinitely.
	if s.DeactivationHeight == 0 {
		return true
	}

	// Deactivation height reached or passed: no longer active.
	if s.DeactivationHeight <= queryHeight {
		return false
	}

	return true
}

// GetActiveServiceConfigs returns all application service configurations that
// are active at the given block height.
//
// This is the historical-aware accessor used by session hydration: a session for
// a past height must resolve the service config that was active at that height,
// not whatever the latest restake overwrote.
//
// Empty history is meaningful: it means the application has never changed its
// service configuration, so its flat ServiceConfigs snapshot has been its config
// for its entire staked lifetime and is returned as active for any height. This
// is why service_config_history is only written when a config actually changes
// (and why no migration/backfill is required for already-staked apps). History
// is only consulted once the app has changed at least once, at which point it
// records the full timeline including the current (still-active) entry.
func (s *Application) GetActiveServiceConfigs(
	queryHeight int64,
) []*sharedtypes.ApplicationServiceConfig {
	// No recorded changes: the flat snapshot is the config for all heights.
	if len(s.ServiceConfigHistory) == 0 {
		return s.ServiceConfigs
	}

	activeServiceConfigs := make([]*sharedtypes.ApplicationServiceConfig, 0)
	for _, serviceConfigUpdate := range s.ServiceConfigHistory {
		if serviceConfigUpdate.IsActive(queryHeight) {
			activeServiceConfigs = append(activeServiceConfigs, serviceConfigUpdate.Service)
		}
	}
	return activeServiceConfigs
}

// BackfillServiceConfigHistory populates an empty service_config_history from the
// application's flat ServiceConfigs snapshot, marking each config active since
// genesis (activation height 1, no deactivation).
//
// It is idempotent: a no-op when history already exists or there are no flat
// configs to backfill. Returns true if the application was modified.
//
// Used by both the v0.1.34 upgrade migration and genesis import so that every
// already-staked application has a resolvable historical config (activation
// height 1 guarantees historical session queries at any past staked height
// return the config, since no real pre-history swap data exists to reconstruct).
func (s *Application) BackfillServiceConfigHistory() bool {
	if len(s.ServiceConfigHistory) > 0 {
		return false
	}
	if len(s.ServiceConfigs) == 0 {
		return false
	}

	history := make([]*ApplicationServiceConfigUpdate, 0, len(s.ServiceConfigs))
	for _, svc := range s.ServiceConfigs {
		history = append(history, &ApplicationServiceConfigUpdate{
			ApplicationAddress: s.Address,
			Service:            svc,
			ActivationHeight:   1,
			DeactivationHeight: 0,
		})
	}
	s.ServiceConfigHistory = history

	return true
}

// GetApplicationUnbondingHeight returns the session end height at which the given
// application finishes unbonding.
func GetApplicationUnbondingHeight(
	sharedParams *sharedtypes.Params,
	application *Application,
) int64 {
	applicationUnbondingPeriodBlocks := sharedParams.ApplicationUnbondingPeriodSessions * sharedParams.NumBlocksPerSession

	return int64(application.UnstakeSessionEndHeight + applicationUnbondingPeriodBlocks)
}

// GetApplicationTransferHeight returns the session end height at which the given
// application transfer completes.
func GetApplicationTransferHeight(
	sharedParams *sharedtypes.Params,
	application *Application,
) int64 {
	pendingTransferSessionEnd := int64(application.GetPendingTransfer().GetSessionEndHeight())

	return sharedtypes.GetSettlementSessionEndHeight(sharedParams, pendingTransferSessionEnd)
}
