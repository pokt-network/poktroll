package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	proto "github.com/cosmos/gogoproto/proto"
)

var nonIndexedEventFields = map[string][]string{
	"pocket.gateway.EventGatewayStaked": {
		"gateway", "session_end_height",
	},
	"pocket.gateway.EventGatewayUnbondingBegin": {
		"gateway", "unbonding_end_height",
	},
	"pocket.gateway.EventGatewayUnbondingEnd": {
		"gateway", "unbonding_end_height",
	},
	"pocket.gateway.EventGatewayUnbondingCanceled": {
		"gateway",
	},
}

func EmitEventGatewayStaked(ctx context.Context, event *EventGatewayStaked) error {
	return EmitTypedEventWithDefaults(ctx, event)
}

func EmitEventGatewayUnbondingBegin(ctx context.Context, event *EventGatewayUnbondingBegin) error {
	return EmitTypedEventWithDefaults(ctx, event)
}

func EmitEventGatewayUnbondingEnd(ctx context.Context, event *EventGatewayUnbondingEnd) error {
	return EmitTypedEventWithDefaults(ctx, event)
}

func EmitEventGatewayUnbondingCanceled(ctx context.Context, event *EventGatewayUnbondingCanceled) error {
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