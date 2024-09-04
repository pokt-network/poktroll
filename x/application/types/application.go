package types

import sharedtypes "github.com/pokt-network/poktroll/x/shared/types"

// ApplicationNotUnstaking is the value of `unstake_session_end_height` if the
// application is not actively in the unbonding period.
const ApplicationNotUnstaking uint64 = 0

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
func (s *Application) IsActive(queryHeight int64) bool {
	return !s.IsUnbonding() || !s.HasPendingTransfer() ||
		uint64(queryHeight) <= s.GetUnstakeSessionEndHeight() ||
		uint64(queryHeight) <= s.GetPendingTransfer().GetSessionEndHeight()
}

// GetApplicationUnbondingHeight returns the session end height at which the given
// application finishes unbonding.
func GetApplicationUnbondingHeight(
	sharedParams *sharedtypes.Params,
	application *Application,
) int64 {
	applicationUnbondingPeriodBlocks := sharedParams.ApplicationTransferAndUnbondingPeriodSessions * sharedParams.NumBlocksPerSession

	return int64(application.UnstakeSessionEndHeight + applicationUnbondingPeriodBlocks)
}

// GetApplicationTransferHeight returns the session end height at which the given
// application finishes transferring. It returns -1 if the application has no
// pending transfer.
func GetApplicationTransferHeight(
	sharedParams *sharedtypes.Params,
	application *Application,
) int64 {
	// TODO_IN_THIS_COMMIT: add and use a new param: `ApplicationTransferPeriodSessions`.
	applicationTransferPeriodBlocks := sharedParams.ApplicationTransferAndUnbondingPeriodSessions * sharedParams.NumBlocksPerSession

	pendingTransfer := application.GetPendingTransfer()
	if pendingTransfer == nil {
		return -1
	}

	return int64(pendingTransfer.GetSessionEndHeight() + applicationTransferPeriodBlocks)
}
