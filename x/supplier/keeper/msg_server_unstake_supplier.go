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
	supplier, isSupplierFound := k.GetSupplier(ctx, msg.OperatorAddress)
	if !isSupplierFound {
		logger.Info(fmt.Sprintf("Supplier not found. Cannot unstake address %s", msg.OperatorAddress))
		return nil, types.ErrSupplierNotFound
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
		return nil, types.ErrSupplierIsUnstaking
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
	supplier.UnstakeSessionEndHeight = uint64(shared.GetSessionEndHeight(&sharedParams, currentHeight))
	k.SetSupplier(ctx, supplier)

	isSuccessful = true
	return &types.MsgUnstakeSupplierResponse{}, nil
}
