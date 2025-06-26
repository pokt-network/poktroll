package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	proto "github.com/cosmos/gogoproto/proto"
)

var nonIndexedEventFields = map[string][]string{
	"pocket.migration.EventImportMorseClaimableAccounts": {
		"morse_account_state_hash", "num_accounts",
	},
	"pocket.migration.EventMorseAccountClaimed": {
		"claimed_balance", "shannon_dest_address", "morse_src_address",
	},
	"pocket.migration.EventMorseApplicationClaimed": {
		"claimed_balance", "morse_src_address", "claimed_application_stake", "application",
	},
	"pocket.migration.EventMorseSupplierClaimed": {
		"claimed_balance", "morse_node_address", "morse_output_address", 
		"claim_signer_type", "claimed_supplier_stake", "supplier",
	},
	"pocket.migration.EventMorseAccountRecovered": {
		"recovered_balance", "shannon_dest_address", "morse_src_address",
	},
}

func EmitEventImportMorseClaimableAccounts(ctx context.Context, event *EventImportMorseClaimableAccounts) error {
	return EmitTypedEventWithDefaults(ctx, event)
}

func EmitEventMorseAccountClaimed(ctx context.Context, event *EventMorseAccountClaimed) error {
	return EmitTypedEventWithDefaults(ctx, event)
}

func EmitEventMorseApplicationClaimed(ctx context.Context, event *EventMorseApplicationClaimed) error {
	return EmitTypedEventWithDefaults(ctx, event)
}

func EmitEventMorseSupplierClaimed(ctx context.Context, event *EventMorseSupplierClaimed) error {
	return EmitTypedEventWithDefaults(ctx, event)
}

func EmitEventMorseAccountRecovered(ctx context.Context, event *EventMorseAccountRecovered) error {
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