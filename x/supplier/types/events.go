package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	proto "github.com/cosmos/gogoproto/proto"
)

var nonIndexedEventFields = map[string][]string{
	"pocket.supplier.EventSupplierStaked": {
		"supplier", "session_end_height",
	},
	"pocket.supplier.EventSupplierUnbondingBegin": {
		"supplier", "reason", "unbonding_end_height",
	},
	"pocket.supplier.EventSupplierUnbondingEnd": {
		"supplier", "reason", "unbonding_end_height",
	},
	"pocket.supplier.EventSupplierUnbondingCanceled": {
		"supplier", "height",
	},
	"pocket.supplier.EventSupplierServiceConfigActivated": {
		"supplier", "activation_height",
	},
}

func EmitEventSupplierStaked(ctx context.Context, event *EventSupplierStaked) error {
	return EmitTypedEventWithDefaults(ctx, event)
}

func EmitEventSupplierUnbondingBegin(ctx context.Context, event *EventSupplierUnbondingBegin) error {
	return EmitTypedEventWithDefaults(ctx, event)
}

func EmitEventSupplierUnbondingEnd(ctx context.Context, event *EventSupplierUnbondingEnd) error {
	return EmitTypedEventWithDefaults(ctx, event)
}

func EmitEventSupplierUnbondingCanceled(ctx context.Context, event *EventSupplierUnbondingCanceled) error {
	return EmitTypedEventWithDefaults(ctx, event)
}

func EmitEventSupplierServiceConfigActivated(ctx context.Context, event *EventSupplierServiceConfigActivated) error {
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