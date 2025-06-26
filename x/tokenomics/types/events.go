package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	proto "github.com/cosmos/gogoproto/proto"
)

var nonIndexedEventFields = map[string][]string{
	"pocket.tokenomics.EventClaimExpired": {
		"claim", "expiration_reason", "num_relays", "num_claimed_compute_units", 
		"num_estimated_compute_units", "claimed_upokt",
	},
	"pocket.tokenomics.EventClaimSettled": {
		"claim", "proof_requirement", "num_relays", "num_claimed_compute_units", 
		"num_estimated_compute_units", "claimed_upokt",
	},
	"pocket.tokenomics.EventApplicationOverserviced": {
		"expected_burn", "effective_burn",
	},
	"pocket.tokenomics.EventSupplierSlashed": {
		"claim", "proof_missing_penalty",
	},
	"pocket.tokenomics.EventClaimDiscarded": {
		"claim", "error",
	},
	"pocket.tokenomics.EventApplicationReimbursementRequest": {
		"amount",
	},
}

func EmitEventClaimExpired(ctx context.Context, event *EventClaimExpired) error {
	return EmitTypedEventWithDefaults(ctx, event)
}

func EmitEventClaimSettled(ctx context.Context, event *EventClaimSettled) error {
	return EmitTypedEventWithDefaults(ctx, event)
}

func EmitEventApplicationOverserviced(ctx context.Context, event *EventApplicationOverserviced) error {
	return EmitTypedEventWithDefaults(ctx, event)
}

func EmitEventSupplierSlashed(ctx context.Context, event *EventSupplierSlashed) error {
	return EmitTypedEventWithDefaults(ctx, event)
}

func EmitEventClaimDiscarded(ctx context.Context, event *EventClaimDiscarded) error {
	return EmitTypedEventWithDefaults(ctx, event)
}

func EmitEventApplicationReimbursementRequest(ctx context.Context, event *EventApplicationReimbursementRequest) error {
	return EmitTypedEventWithDefaults(ctx, event)
}

func EmitTypedEventWithDefaults(ctx context.Context, msg proto.Message) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	
	event, err := sdk.TypedEventToEvent(msg)
	if err != nil {
		return err
	}

	if nonIndexedKeys, exists := nonIndexedEventFields[event.Type]; exists {
		for i, attr := range event.Attributes {
			for _, nonIndexedKey := range nonIndexedKeys {
				if attr.Key == nonIndexedKey {
					event.Attributes[i].Index = false
					break
				}
			}
		}
	}

	sdkCtx.EventManager().EmitEvent(event)
	return nil
}