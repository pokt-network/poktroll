package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/pocket/telemetry"
	sharedtypes "github.com/pokt-network/pocket/x/shared/types"
	suppliertypes "github.com/pokt-network/pocket/x/supplier/types"
)

func (k msgServer) UnstakeSupplier(
	ctx context.Context,
	msg *suppliertypes.MsgUnstakeSupplier,
) (*suppliertypes.MsgUnstakeSupplierResponse, error) {
	isSuccessful := false
	defer telemetry.EventSuccessCounter(
		"unstake_supplier",
		telemetry.DefaultCounterFn,
		func() bool { return isSuccessful },
	)

	logger := k.Logger().With("method", "UnstakeSupplier")
	logger.Info(fmt.Sprintf("About to unstake supplier with msg: %v", msg))

	if err := msg.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Check if the supplier already exists or not
	supplier, isSupplierFound := k.GetSupplier(ctx, msg.GetOperatorAddress())
	if !isSupplierFound {
		logger.Info(fmt.Sprintf("Supplier not found. Cannot unstake address %s", msg.GetOperatorAddress()))
		return nil, status.Error(
			codes.NotFound,
			suppliertypes.ErrSupplierNotFound.Wrapf(
				"supplier with operator address %q", msg.GetOperatorAddress(),
			).Error(),
		)
	}

	// Ensure the singer address matches the owner address or the operator address.
	if !supplier.HasOperator(msg.GetSigner()) && !supplier.HasOwner(msg.GetSigner()) {
		logger.Info("only the supplier owner or operator is allowed to unstake the supplier")
		return nil, status.Error(
			codes.PermissionDenied,
			sharedtypes.ErrSharedUnauthorizedSupplierUpdate.Wrapf(
				"signer %q is not allowed to unstake supplier %+v",
				msg.Signer,
				supplier,
			).Error(),
		)
	}

	logger.Info(fmt.Sprintf("Supplier found. Unstaking supplier with operating address %s", msg.GetOperatorAddress()))

	// Check if the supplier has already initiated the unstake action.
	if supplier.IsUnbonding() {
		logger.Info(fmt.Sprintf("Supplier %s still unbonding from previous unstaking", msg.GetOperatorAddress()))
		return nil, status.Error(
			codes.FailedPrecondition,
			suppliertypes.ErrSupplierIsUnstaking.Wrapf(
				"supplier with operator address %q", msg.GetOperatorAddress(),
			).Error(),
		)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()
	sharedParams := k.sharedKeeper.GetParams(ctx)

	// Mark the supplier as unstaking by recording the height at which it should stop
	// providing service.
	// The supplier MUST continue to provide service until the end of the current
	// session. I.e., onchain sessions' suppliers list MUST NOT change mid-session.
	// Removing it right away could have undesired effects on the network
	// (e.g. a session with less than the minimum or 0 number of suppliers,
	// offchain actors that need to listen to session supplier's change mid-session, etc).
	supplier.UnstakeSessionEndHeight = uint64(sharedtypes.GetSessionEndHeight(&sharedParams, currentHeight))

	// Update the supplier's service config history with the empty service configs
	// to indicate that the supplier is no longer providing service after the current session.
	nextSessionStartHeight := sharedtypes.GetNextSessionStartHeight(&sharedParams, currentHeight)
	servicesUpdate := &sharedtypes.ServiceConfigUpdate{
		Services:             make([]*sharedtypes.SupplierServiceConfig, 0),
		EffectiveBlockHeight: uint64(nextSessionStartHeight),
	}

	serviceConfigUpdateList := supplier.ServiceConfigHistory
	serviceConfigLatestUpdateIdx := len(serviceConfigUpdateList) - 1
	// Overwrite the latest service configuration if there is already a service
	// config update for the same session start height.
	// This is to avoid having duplicate service configs with the same activation
	// height, which is useless and potentially confusing.
	if serviceConfigLatestUpdateIdx >= 0 && serviceConfigUpdateList[serviceConfigLatestUpdateIdx].EffectiveBlockHeight == uint64(nextSessionStartHeight) {
		supplier.ServiceConfigHistory[serviceConfigLatestUpdateIdx] = servicesUpdate
	} else {
		// Otherwise, append the new service configuration update.
		supplier.ServiceConfigHistory = append(supplier.ServiceConfigHistory, servicesUpdate)
	}

	k.SetSupplier(ctx, supplier)

	// Emit an event which signals that the supplier successfully began unbonding their stake.
	unbondingEndHeight := sharedtypes.GetSupplierUnbondingEndHeight(&sharedParams, &supplier)
	event := &suppliertypes.EventSupplierUnbondingBegin{
		Supplier:           &supplier,
		Reason:             suppliertypes.SupplierUnbondingReason_SUPPLIER_UNBONDING_REASON_VOLUNTARY,
		SessionEndHeight:   int64(supplier.GetUnstakeSessionEndHeight()),
		UnbondingEndHeight: unbondingEndHeight,
	}
	if err := sdkCtx.EventManager().EmitTypedEvent(event); err != nil {
		err = suppliertypes.ErrSupplierEmitEvent.Wrapf("(%+v): %s", event, err)
		logger.Error(err.Error())
		return nil, status.Error(codes.Internal, err.Error())
	}

	isSuccessful = true
	return &suppliertypes.MsgUnstakeSupplierResponse{
		Supplier: &supplier,
	}, nil
}
