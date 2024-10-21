package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/cometbft/cometbft/rpc/client/http"
)

// This is a poktrolld (cosmos-sdk) prometheus metrics exporter. It gets the data from on-chain information
// and exposes them as prometheus metrics.

// https://chatgpt.com/c/6712e754-2af4-8006-b9a9-1cdf5351b808

// Future todos:
// - Add a root struct that holds latest block. When the latest block is updated, call some functions to update the metrics.
// - Utilize events to update some metrics instead of requesting the data from on-chain.
// - Look at https://github.com/PFC-developer/cosmos-exporter for ideas on additional metrics to expose:
//   - Expose params. Use labels: `module`, `param_name`. Don't copy from above as that implementation is not what we need.
//   - Look at validators.
//   - Copy upgrades.
//   - Copy wallets.
//   - Copy basic metrics.

func main() {
	client, err := http.New("tcp://127.0.0.1:26657", "/websocket")
	if err != nil {
		log.Fatal("can't connect", err)
	}

	err = client.Start()
	if err != nil {
		log.Fatal(err)
	}
	defer client.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	query := "tm.event = 'NewBlock'"
	txs, err := client.Subscribe(ctx, "test-client", query)
	if err != nil {
		log.Fatal(err)
	}

	for e := range txs {
		switch data := e.Data.(type) {
		case types.EventDataNewBlock:
			// TODO: Have a root struct that holds latest block.
			// When the block is updated -> we should also call some functions to update the metric values.
			fmt.Printf("EventDataNewBlock: Block %s - Height: %d \n", hex.EncodeToString(data.Block.Hash()), data.Block.Height)
			for _, event := range data.ResultFinalizeBlock.Events {
				// skip event if event.Type doesn't start with "poktroll."
				if !strings.HasPrefix(event.Type, "poktroll.") {
					continue
				}

				// Quote the event attribute values
				quoteEventAttributes(&event)

				// for _, attr := range event.Attributes {
				// 	fmt.Printf("Attribute: Key=%s, Value=%s\n", attr.Key, attr.Value)
				// }

				parsedEvent, err := cosmostypes.ParseTypedEvent(event)
				if err != nil {
					fmt.Printf("failed to parse event of type %s: %v\n", event.Type, err)
					continue
				}

				switch e := parsedEvent.(type) {
				case *prooftypes.EventProofUpdated:
					fmt.Printf("EventProofUpdated: %s, %d, %d\n", e.ClaimedUpokt, e.NumClaimedComputeUnits, e.NumEstimatedComputeUnits)
				case *prooftypes.EventClaimCreated:
					fmt.Printf("EventClaimCreated: %s, %d, %d\n", e.ClaimedUpokt, e.NumClaimedComputeUnits, e.NumEstimatedComputeUnits)
				case *tokenomicstypes.EventClaimSettled:
					fmt.Printf("EventClaimSettled: %s, %d, %d\n", e.ClaimedUpokt, e.NumClaimedComputeUnits, e.NumEstimatedComputeUnits)
				case *tokenomicstypes.EventClaimExpired:
					fmt.Printf("EventClaimExpired (%s): %s, %d, %d\n", e.ExpirationReason, e.ClaimedUpokt, e.NumClaimedComputeUnits, e.NumEstimatedComputeUnits)
				case *tokenomicstypes.EventSupplierSlashed:
					fmt.Printf("EventSupplierSlashed (%s) - %s", e.SupplierOperatorAddr, e.SlashingAmount)
				case *servicetypes.EventRelayMiningDifficultyUpdated:
					fmt.Printf("EventRelayMiningDifficultyUpdated: %s, %d->%d, %s->%s\n", e.ServiceId, e.PrevNumRelaysEma, e.NewNumRelaysEma, e.PrevTargetHashHexEncoded, e.NewTargetHashHexEncoded)
				default:
					fmt.Printf("Unknown event type: %s\n", event.Type)
				}
			}

		}
	}

}

func quoteEventAttributes(event *abci.Event) {
	for i, attr := range event.Attributes {
		var js json.RawMessage
		if err := json.Unmarshal([]byte(attr.Value), &js); err != nil {
			// Value is not valid JSON, so wrap it in quotes
			event.Attributes[i].Value = fmt.Sprintf("%q", attr.Value)
		} else {
			// Value is valid JSON, leave it as is
			event.Attributes[i].Value = attr.Value
		}
	}
}
