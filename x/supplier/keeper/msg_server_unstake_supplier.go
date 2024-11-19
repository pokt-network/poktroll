package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/telemetry"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
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
		return nil, err
	}

	// Check if the supplier already exists or not
	supplier, isSupplierFound := k.GetSupplier(ctx, msg.OperatorAddress)
	if !isSupplierFound {
		logger.Info(fmt.Sprintf("Supplier not found. Cannot unstake address %s", msg.OperatorAddress))
		return nil, suppliertypes.ErrSupplierNotFound
	}

	// Ensure the singer address matches the owner address or the operator address.
	if !supplier.HasOperator(msg.Signer) && !supplier.HasOwner(msg.Signer) {
		logger.Error("only the supplier owner or operator is allowed to unstake the supplier")
		return nil, sharedtypes.ErrSharedUnauthorizedSupplierUpdate.Wrapf(
			"signer %q is not allowed to unstake supplier %v",
			msg.Signer,
			supplier,
		)
	}

	logger.Info(fmt.Sprintf("Supplier found. Unstaking supplier for address %s", msg.OperatorAddress))

	// Check if the supplier has already initiated the unstake action.
	if supplier.IsUnbonding() {
		logger.Warn(fmt.Sprintf("Supplier %s still unbonding from previous unstaking", msg.OperatorAddress))
		return nil, suppliertypes.ErrSupplierIsUnstaking
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()
	sharedParams := k.sharedKeeper.GetParams(ctx)

	// Mark the supplier as unstaking by recording the height at which it should stop
	// providing service.
	// The supplier MUST continue to provide service until the end of the current
	// session. I.e., on-chain sessions' suppliers list MUST NOT change mid-session.
	// Removing it right away could have undesired effects on the network
	// (e.g. a session with less than the minimum or 0 number of suppliers,
	// off-chain actors that need to listen to session supplier's change mid-session, etc).
	supplier.UnstakeSessionEndHeight = uint64(sharedtypes.GetSessionEndHeight(&sharedParams, currentHeight))
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
	return &suppliertypes.MsgUnstakeSupplierResponse{}, nil
}
