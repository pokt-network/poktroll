package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/telemetry"
	"github.com/pokt-network/poktroll/x/shared"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

// TODO_BETA(#489): Determine if an application needs an unbonding period after unstaking.
func (k msgServer) UnstakeSupplier(
	ctx context.Context,
	msg *types.MsgUnstakeSupplier,
) (*types.MsgUnstakeSupplierResponse, error) {
	isSuccessful := false
	defer telemetry.EventSuccessCounter(
		"unstake_supplier",
		telemetry.DefaultCounterFn,
		func() bool { return isSuccessful },
	)

	logger := k.Logger().With("method", "UnstakeSupplier")
	logger.Info(fmt.Sprintf("About to unstake supplier with msg: %v", msg))

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// Check if the supplier already exists or not
	supplier, isSupplierFound := k.GetSupplier(ctx, msg.Address)
	if !isSupplierFound {
		logger.Info(fmt.Sprintf("Supplier not found. Cannot unstake address %s", msg.Address))
		return nil, types.ErrSupplierNotFound
	}
	logger.Info(fmt.Sprintf("Supplier found. Unstaking supplier for address %s", msg.Address))

	// Check if the supplier has already initiated the unstake action and is in the unbonding period
	if supplier.UnbondingHeight > 0 {
		logger.Warn(fmt.Sprintf("Supplier %s has not finished the unbonding period", msg.Address))
		return nil, types.ErrSupplierUnbonding
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()
	sharedParams := k.sharedKeeper.GetParams(ctx)

	supplier.UnbondingHeight = GetSupplierUnbondingHeight(&sharedParams, currentHeight)
	k.SetSupplier(ctx, supplier)

	isSuccessful = true
	return &types.MsgUnstakeSupplierResponse{}, nil
}

// GetSupplierUnbondingHeight returns the height at which the supplier will be able to withdraw
// the staked coins after the unbonding period.
func GetSupplierUnbondingHeight(sharedParams *sharedtypes.Params, currentHeight int64) int64 {
	sessionEndHeight := shared.GetSessionEndHeight(sharedParams, currentHeight)

	// TODO_IN_THIS_PR: Make the unbonding period a governance parameter.
	unbondingPeriodInBlocks := int64(sharedParams.GetNumBlocksPerSession())

	// Unbonding period has a minimum duration of 1 session, which means that if
	// the current height is prior to the end of the session, the unbonding height
	// will be set to the end of the session that is after the current session.
	// This is to avoid the case where a supplier is able to withdraw after 1 block,
	// if it unstakes right before the end of the current session.
	return sessionEndHeight + unbondingPeriodInBlocks

}
