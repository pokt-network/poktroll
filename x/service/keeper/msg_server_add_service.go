package keeper

import (
	"bytes"
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/telemetry"
	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// AddService adds a service to the network.
// The operation checks if the signer has enough funds (upokt) to pay the AddServiceFee.
// If funds are insufficient, the service won't be added. Otherwise, the fee is transferred from
// the signer to the service module's account, afterwards the service will be present onchain.
func (k msgServer) AddService(
	goCtx context.Context,
	msg *types.MsgAddService,
) (*types.MsgAddServiceResponse, error) {
	isSuccessful := false
	defer telemetry.EventSuccessCounter(
		"add_service",
		telemetry.DefaultCounterFn,
		func() bool { return isSuccessful },
	)

	ctx := sdk.UnwrapSDKContext(goCtx)

	logger := k.Logger().With("method", "AddService")
	// Identifying fields only, NEVER the whole message: Service carries a metadata card
	// of up to 256 KiB, and rendering it inflates the line to ~317 KB -- on every
	// validator and full node, for every AddService.
	logger.Info(fmt.Sprintf(
		"About to add a new service with id: %q (owner: %s)",
		msg.Service.GetId(), msg.GetOwnerAddress(),
	))

	// Validate the message.
	if err := msg.ValidateBasic(); err != nil {
		logger.Error(fmt.Sprintf("Adding service failed basic validation: %v", err))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Check if the service already exists or not.
	foundService, found := k.GetService(ctx, msg.Service.Id)
	if found {
		// Verify the owner address matches the existing service owner
		if foundService.OwnerAddress != msg.Service.OwnerAddress {
			return nil, status.Error(
				codes.InvalidArgument,
				types.ErrServiceInvalidOwnerAddress.Wrapf(
					"existing owner address %q does not match the new owner address %q",
					foundService.OwnerAddress, msg.Service.OwnerAddress,
				).Error(),
			)
		}

		// TODO_POST_MIGRATION: This logic will be replaced once the following PR is merged:
		// https://github.com/pokt-network/poktroll/pull/1388

		// Update the service fields that are allowed to change
		logger.Info(fmt.Sprintf("Updating service: ComputeUnitsPerRelay=%v, HasMetadata=%v",
			msg.Service.ComputeUnitsPerRelay, msg.Service.Metadata != nil))

		// Capture the previous cupr before overwriting so a change can be snapshotted
		// for session-start-pinned claim validation.
		prevComputeUnitsPerRelay := foundService.ComputeUnitsPerRelay

		// Only overwrite metadata when the message actually carries a DIFFERENT card.
		//
		// MsgAddService is the ONLY update path for an existing service and always
		// carries a full Service{}, so a message that only intends to change
		// compute_units_per_relay still arrives with a nil Metadata. Assigning it
		// unconditionally would silently destroy the stored metadata of every service
		// updated by a client that does not re-send it (e.g. the `edit-service` CLI,
		// which builds its message via NewMsgAddService and never sets Metadata).
		//
		// TODO_TECHDEBT: An owner can ERASE metadata by submitting a minimal payload
		// (e.g. `{}`), which overwrites the stored blob, but cannot set the field back to
		// nil: Metadata.ValidateBasic rejects a zero-length card payload, so a
		// non-nil Metadata always remains. Consumers must therefore treat a non-nil
		// Metadata as "may be empty", not "has content". Add a dedicated clear mechanism
		// if/when the metadata fields are reworked.
		isCardChanged := msg.Service.Metadata != nil &&
			!bytes.Equal(foundService.Metadata.GetCard(), msg.Service.Metadata.GetCard())

		// Short-circuit a no-op update: nothing to write, nothing to announce.
		//
		// cosmos-sdk's KVStore marks the key dirty on any Set, identical value or not, so
		// re-Setting an unchanged Service creates a fresh IAVL node at commit -- a full
		// extra copy of a card up to MaxServiceMetadataSizeBytes, retained forever by
		// archive nodes. Re-broadcasting the same service would then be an unbounded
		// state-growth lever costing only gas. Same guard as UpdateGatewayMetadata.
		if !isCardChanged &&
			foundService.Name == msg.Service.Name &&
			prevComputeUnitsPerRelay == msg.Service.ComputeUnitsPerRelay {
			logger.Info(fmt.Sprintf("Service %q is already up to date; skipping redundant write", msg.Service.Id))
			isSuccessful = true
			return &types.MsgAddServiceResponse{}, nil
		}

		foundService.Name = msg.Service.Name
		foundService.ComputeUnitsPerRelay = msg.Service.ComputeUnitsPerRelay
		if isCardChanged {
			foundService.Metadata = msg.Service.Metadata
		}

		k.SetService(ctx, foundService)

		// Record the cupr change in history so claim validation resolves the cupr that
		// was live at each session's start (in-flight sessions keep their start rate;
		// the new value takes effect at the next session boundary). Only record an
		// actual change — name/metadata-only updates leave cupr history untouched.
		if prevComputeUnitsPerRelay != msg.Service.ComputeUnitsPerRelay {
			if err := k.SnapshotServiceComputeUnitsPerRelayChange(
				ctx,
				msg.Service.Id,
				prevComputeUnitsPerRelay,
				msg.Service.ComputeUnitsPerRelay,
			); err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
		}

		if isCardChanged {
			if err := k.emitServiceMetadataUpdated(ctx, foundService); err != nil {
				return nil, err
			}
		}

		isSuccessful = true
		return &types.MsgAddServiceResponse{}, nil
	}

	// Retrieve the address of the actor adding the service; the owner of the service.
	serviceOwnerAddr, err := sdk.AccAddressFromBech32(msg.OwnerAddress)
	if err != nil {
		return nil, status.Error(
			codes.InvalidArgument,
			types.ErrServiceInvalidAddress.Wrapf(
				"%s is not in Bech32 format", msg.OwnerAddress,
			).Error(),
		)
	}

	// Check the actor has sufficient funds to pay for the add service fee.
	accCoins := k.bankKeeper.SpendableCoins(ctx, serviceOwnerAddr)
	if accCoins.Len() == 0 {
		return nil, status.Error(
			codes.FailedPrecondition,
			types.ErrServiceNotEnoughFunds.Wrapf(
				"account has no spendable coins",
			).Error(),
		)
	}

	// Check the balance of upokt is enough to cover the AddServiceFee.
	accBalance := accCoins.AmountOf("upokt")
	addServiceFee := k.GetParams(ctx).AddServiceFee
	if accBalance.LTE(addServiceFee.Amount) {
		return nil, status.Error(
			codes.FailedPrecondition,
			types.ErrServiceNotEnoughFunds.Wrapf(
				"account has %s, but the service fee is %s",
				accBalance, k.GetParams(ctx).AddServiceFee,
			).Error(),
		)
	}

	// Deduct the service fee from the actor's balance.
	serviceFee := sdk.NewCoins(*addServiceFee)
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, serviceOwnerAddr, types.ModuleName, serviceFee)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to deduct service fee from actor's balance: %+v", err))
		return nil, status.Error(
			codes.Internal,
			types.ErrServiceFailedToDeductFee.Wrapf(
				"account has %s, failed to deduct %s",
				accBalance, k.GetParams(ctx).AddServiceFee,
			).Error(),
		)
	}

	// See the note above: never render the Service value, it embeds the metadata card.
	logger.Info(fmt.Sprintf(
		"Adding service id: %q (compute units per relay: %d, has metadata: %t)",
		msg.Service.GetId(), msg.Service.GetComputeUnitsPerRelay(), msg.Service.GetMetadata() != nil,
	))
	k.SetService(ctx, msg.Service)

	// Seed the initial cupr in history so future changes have a baseline and
	// claim validation can resolve the session-start cupr for this service.
	if err := k.SnapshotServiceComputeUnitsPerRelayCreate(
		ctx,
		msg.Service.Id,
		msg.Service.ComputeUnitsPerRelay,
	); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// A new service created with a card announces it too: from an indexer's point of
	// view the stored card changed (from none to this one).
	if msg.Service.Metadata != nil {
		if err := k.emitServiceMetadataUpdated(ctx, msg.Service); err != nil {
			return nil, err
		}
	}

	isSuccessful = true
	return &types.MsgAddServiceResponse{}, nil
}

// emitServiceMetadataUpdated emits EventServiceMetadataUpdated for service, whose
// card was just stored. The event carries the card's size, never the card itself.
func (k msgServer) emitServiceMetadataUpdated(ctx sdk.Context, service sharedtypes.Service) error {
	sharedParams := k.sharedKeeper.GetParams(ctx)
	event := &types.EventServiceMetadataUpdated{
		ServiceId:        service.Id,
		SessionEndHeight: sharedtypes.GetSessionEndHeight(&sharedParams, ctx.BlockHeight()),
		CardSizeBytes:    uint64(len(service.Metadata.GetCard())),
	}
	if err := ctx.EventManager().EmitTypedEvent(event); err != nil {
		err = types.ErrServiceEmitEvent.Wrapf("(%+v): %s", event, err)
		k.Logger().Error(err.Error())
		return status.Error(codes.Internal, err.Error())
	}
	return nil
}
