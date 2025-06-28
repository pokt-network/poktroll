package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	proto "github.com/cosmos/gogoproto/proto"
)

var nonIndexedEventFields = map[string][]string{
	"pocket.application.EventApplicationStaked": {
		"application", "session_end_height",
	},
	"pocket.application.EventRedelegation": {
		"application", "session_end_height",
	},
	"pocket.application.EventTransferBegin": {
		"source_application", "transfer_end_height",
	},
	"pocket.application.EventTransferEnd": {
		"destination_application", "transfer_end_height",
	},
	"pocket.application.EventTransferError": {
		"source_application", "error",
	},
	"pocket.application.EventApplicationUnbondingBegin": {
		"application", "unbonding_end_height",
	},
	"pocket.application.EventApplicationUnbondingEnd": {
		"application", "unbonding_end_height",
	},
	"pocket.application.EventApplicationUnbondingCanceled": {
		"application",
	},
}

func EmitEventApplicationStaked(ctx context.Context, event *EventApplicationStaked) error {
	return EmitTypedEventWithDefaults(ctx, event)
}

func EmitEventRedelegation(ctx context.Context, event *EventRedelegation) error {
	return EmitTypedEventWithDefaults(ctx, event)
}

func EmitEventTransferBegin(ctx context.Context, event *EventTransferBegin) error {
	return EmitTypedEventWithDefaults(ctx, event)
}

func EmitEventTransferEnd(ctx context.Context, event *EventTransferEnd) error {
	return EmitTypedEventWithDefaults(ctx, event)
}

func EmitEventTransferError(ctx context.Context, event *EventTransferError) error {
	return EmitTypedEventWithDefaults(ctx, event)
}

func EmitEventApplicationUnbondingBegin(ctx context.Context, event *EventApplicationUnbondingBegin) error {
	return EmitTypedEventWithDefaults(ctx, event)
}

func EmitEventApplicationUnbondingEnd(ctx context.Context, event *EventApplicationUnbondingEnd) error {
	return EmitTypedEventWithDefaults(ctx, event)
}

func EmitEventApplicationUnbondingCanceled(ctx context.Context, event *EventApplicationUnbondingCanceled) error {
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