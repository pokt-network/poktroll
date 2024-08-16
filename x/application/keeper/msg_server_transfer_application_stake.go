package keeper

import (
	"context"
	"fmt"

	"github.com/pokt-network/poktroll/x/application/types"
)

func (k msgServer) TransferApplicationStake(ctx context.Context, msg *types.MsgTransferApplicationStake) (*types.MsgTransferApplicationStakeResponse, error) {
	// TODO_IN_THIS_COMMIT: add telemetry.

	logger := k.Logger().With("method", "TransferApplicationStake")

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	_, isBeneficiaryFound := k.GetApplication(ctx, msg.Beneficiary)
	if isBeneficiaryFound {
		return nil, types.ErrAppDuplicateAddress.Wrapf("beneficiary (%q) exists", msg.Beneficiary)
	}

	foundApp, isAppFound := k.GetApplication(ctx, msg.Address)
	if !isAppFound {
		return nil, types.ErrAppNotFound.Wrapf("application %q not found", msg.Address)
	}

	beneficiary := k.createApplication(ctx, &types.MsgStakeApplication{
		Address:  msg.Beneficiary,
		Stake:    foundApp.Stake,
		Services: foundApp.ServiceConfigs,
	})

	// TODO_TEST: add E2E coverage to assert #DelegateeGatewayAddresses and #PendingUndelegations
	// are present on the beneficiary application.
	beneficiary.DelegateeGatewayAddresses = foundApp.DelegateeGatewayAddresses
	beneficiary.PendingUndelegations = foundApp.PendingUndelegations

	// Update the beneficiary in the store
	k.SetApplication(ctx, beneficiary)
	logger.Info(fmt.Sprintf("Successfully transferred application stake from app (%s) to beneficiary (%s)", foundApp.Address, beneficiary.Address))

	// Remove the transferred app from the store
	k.RemoveApplication(ctx, foundApp.Address)
	logger.Info(fmt.Sprintf("Successfully removed the application: %+v", foundApp))

	return &types.MsgTransferApplicationStakeResponse{
		Application: &beneficiary,
	}, nil
}
