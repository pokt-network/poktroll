package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	proto "github.com/cosmos/gogoproto/proto"
)

var nonIndexedEventFields = map[string][]string{
	"pocket.proof.EventClaimCreated": {
		"claim", "num_relays", "num_claimed_compute_units", 
		"num_estimated_compute_units", "claimed_upokt",
	},
	"pocket.proof.EventClaimUpdated": {
		"claim", "num_relays", "num_claimed_compute_units", 
		"num_estimated_compute_units", "claimed_upokt",
	},
	"pocket.proof.EventProofSubmitted": {
		"claim", "num_relays", "num_claimed_compute_units", 
		"num_estimated_compute_units", "claimed_upokt",
	},
	"pocket.proof.EventProofUpdated": {
		"claim", "num_relays", "num_claimed_compute_units", 
		"num_estimated_compute_units", "claimed_upokt",
	},
	"pocket.proof.EventProofValidityChecked": {
		"claim", "failure_reason",
	},
}

func EmitEventClaimCreated(ctx context.Context, event *EventClaimCreated) error {
	return EmitTypedEventWithDefaults(ctx, event)
}

func EmitEventClaimUpdated(ctx context.Context, event *EventClaimUpdated) error {
	return EmitTypedEventWithDefaults(ctx, event)
}

func EmitEventProofSubmitted(ctx context.Context, event *EventProofSubmitted) error {
	return EmitTypedEventWithDefaults(ctx, event)
}

func EmitEventProofUpdated(ctx context.Context, event *EventProofUpdated) error {
	return EmitTypedEventWithDefaults(ctx, event)
}

func EmitEventProofValidityChecked(ctx context.Context, event *EventProofValidityChecked) error {
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