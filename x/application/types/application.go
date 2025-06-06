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
func (app *Application) IsUnbonding() bool {
	return app.UnstakeSessionEndHeight != ApplicationNotUnstaking
}

// HasPendingTransfer returns true if the application has begun but not completed
// an application transfer. It determines if the application has submitted a transfer
// message, in which case the application has its PendingTransfer field set.
func (app *Application) HasPendingTransfer() bool {
	return app.PendingTransfer != nil
}

// IsActive returns whether the application is allowed to request services at the
// given query height.
// An application that has not submitted an unstake message is always active.
// An application that has submitted an unstake message is active until the end of
// the session containing the height at which unstake message was submitted.
// An application that has a pending transfer is active until the end of the session
// containing the height at which the transfer was initiated.
func (app *Application) IsActive(queryHeight int64) bool {
	return !app.IsUnbonding() || !app.HasPendingTransfer() ||
		uint64(queryHeight) <= app.GetUnstakeSessionEndHeight() ||
		uint64(queryHeight) <= app.GetPendingTransfer().GetSessionEndHeight()
}

// UpdateServiceUsageMetrics increments the service usage metrics for a specific service
// - It finds existing metrics for the service or initializes a new one
// - Increments relay and compute unit counts by the provided values
func (app *Application) UpdateServiceUsageMetrics(
	serviceId string,
	numRelays,
	numComputeUnits uint64,
) {
	serviceUsageMetrics := &sharedtypes.ServiceUsageMetrics{ServiceId: serviceId}

	for _, existingServiceUsageMetrics := range app.ServiceUsageMetrics {
		if existingServiceUsageMetrics.ServiceId == serviceId {
			serviceUsageMetrics = existingServiceUsageMetrics
			break
		}
	}

	// Increment the metrics with the new relay and compute unit counts
	// These values represent the application's total consumption of the service
	serviceUsageMetrics.TotalRelays += numRelays
	serviceUsageMetrics.TotalComputeUnits += numComputeUnits

	app.ServiceUsageMetrics[serviceId] = serviceUsageMetrics
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
	sessionEndToProofWindowCloseBlocks := sharedtypes.GetSessionEndToProofWindowCloseBlocks(sharedParams)

	return int64(application.GetPendingTransfer().GetSessionEndHeight()) + sessionEndToProofWindowCloseBlocks
}
