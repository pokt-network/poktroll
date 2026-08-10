package keeper

import (
	"bytes"
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/telemetry"
	"github.com/pokt-network/poktroll/x/gateway/types"
)

// UpdateGatewayMetadata sets a staked gateway's card WITHOUT touching its stake.
//
// This is deliberately a separate message from MsgStakeGateway, which enforces a
// strictly-positive stake delta on every call: folding metadata into it would mean
// escrowing real POKT to fix a typo in a description.
func (k msgServer) UpdateGatewayMetadata(
	goCtx context.Context,
	msg *types.MsgUpdateGatewayMetadata,
) (*types.MsgUpdateGatewayMetadataResponse, error) {
	isSuccessful := false
	defer telemetry.EventSuccessCounter(
		"update_gateway_metadata",
		telemetry.DefaultCounterFn,
		func() bool { return isSuccessful },
	)

	ctx := sdk.UnwrapSDKContext(goCtx)

	logger := k.Logger().With("method", "UpdateGatewayMetadata")

	// DEV_NOTE: Log the card's SIZE, never the card itself. The other gateway handlers log
	// the whole msg with %v, which is fine for a stake but not here: Metadata.Card can hold
	// up to MaxServiceMetadataSizeBytes, and %v renders a []byte as a bracketed list of
	// decimal numbers -- roughly 4x expansion, so a maxed-out card becomes a ~1 MB single
	// log line on every call.
	logger.Info(fmt.Sprintf(
		"About to update gateway metadata for address %s (card size: %d bytes)",
		msg.Address, len(msg.Metadata.GetCard()),
	))

	if err := msg.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// The gateway must already be staked. Same not-found handling as UnstakeGateway,
	// done here rather than in ValidateBasic, which is stateless.
	gateway, isGatewayFound := k.GetGateway(ctx, msg.Address)
	if !isGatewayFound {
		logger.Info(fmt.Sprintf("Gateway not found. Cannot update metadata for address %s", msg.Address))
		return nil, status.Error(
			codes.NotFound,
			types.ErrGatewayNotFound.Wrapf(
				"gateway with address %s", msg.Address,
			).Error(),
		)
	}

	// Callable regardless of unbonding status: a gateway that is on its way out may still
	// want to correct or withdraw what it advertises.

	// Only overwrite the card when the message actually carries a DIFFERENT one.
	//
	// This mirrors MsgAddService: a nil Metadata means "leave the stored card alone", not
	// "delete it". Treating nil as a clear would let any client that does not re-send the
	// card silently destroy it.
	//
	// TODO_TECHDEBT: As with Service, an owner can ERASE a card by submitting a minimal
	// payload (e.g. `{}`) but cannot set the field back to nil, because
	// Metadata.ValidateBasic rejects a zero-length card. Consumers must treat a non-nil
	// Metadata as "may be empty", not "has content".
	//
	// Short-circuit a no-op update: nothing to write, nothing to announce.
	//
	// Both "leave the stored card alone" (nil metadata) and "set the card to exactly what
	// is already stored" would otherwise re-Set an identical Gateway. cosmos-sdk's KVStore
	// marks the key dirty on any Set, identical value or not, so the commit creates a fresh
	// IAVL node -- a full extra copy of a card up to MaxServiceMetadataSizeBytes, retained
	// forever by archive nodes. Re-broadcasting the same card is then an unbounded
	// state-growth lever costing only gas.
	//
	// Skipping the event too is deliberate: no card changed, so there is nothing for an
	// indexer to apply. The tx still succeeds (it is idempotent, not invalid).
	if msg.Metadata == nil || bytes.Equal(gateway.Metadata.GetCard(), msg.Metadata.GetCard()) {
		logger.Info(fmt.Sprintf(
			"Gateway %s card is already up to date (%d bytes); skipping redundant write",
			msg.Address, len(gateway.Metadata.GetCard()),
		))
		isSuccessful = true
		return &types.MsgUpdateGatewayMetadataResponse{}, nil
	}

	gateway.Metadata = msg.Metadata

	k.SetGateway(ctx, gateway)
	logger.Info(fmt.Sprintf("Successfully updated metadata for gateway: %s", msg.Address))

	sessionEndHeight := k.sharedKeeper.GetSessionEndHeight(ctx, ctx.BlockHeight())
	event := &types.EventGatewayMetadataUpdated{
		GatewayAddress:   gateway.Address,
		SessionEndHeight: int64(sessionEndHeight),
		CardSizeBytes:    uint64(len(gateway.Metadata.GetCard())),
	}
	if err := ctx.EventManager().EmitTypedEvent(event); err != nil {
		err = types.ErrGatewayEmitEvent.Wrapf("(%+v): %s", event, err)
		logger.Error(err.Error())
		return nil, status.Error(codes.Internal, err.Error())
	}

	isSuccessful = true
	return &types.MsgUpdateGatewayMetadataResponse{}, nil
}
