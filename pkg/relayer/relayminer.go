package relayer

import (
	"context"

	"cosmossdk.io/depinject"
)

// relayMiner is the main struct that encapsulates the relayer's responsibilities (i.e. Relay Mining).
// It starts and stops the RelayerProxy and provide the served relays observable to them miner.
type relayMiner struct {
	relayerProxy RelayerProxy
	miner        Miner
}

// NewRelayMiner creates a new Relayer instance with the given dependencies.
// It injects the dependencies into the Relayer instance and returns it.
func NewRelayMiner(deps depinject.Config) (*relayMiner, error) {
	rel := &relayMiner{}

	if err := depinject.Inject(
		deps,
		&rel.relayerProxy,
		&rel.miner,
	); err != nil {
		return nil, err
	}

	return rel, nil
}

// Start provides the miner with the served relays observable and starts the relayer proxy.
// This method is blocking while the relayer proxy is running and returns when Stop is called
// or when the relayer proxy fails to start.
func (rel *relayMiner) Start(ctx context.Context) error {
	// MineRelays does not block and subscribes to the served relays observable.
	rel.miner.StartMiningRelays(ctx, rel.relayerProxy.ServedRelays())
	return rel.relayerProxy.Start(ctx)
}

// Stop stops the relayer proxy which in turn stops all advertised relay servers
// and unsubscribes the miner from the served relays observable.
func (rel *relayMiner) Stop(ctx context.Context) error {
	return rel.relayerProxy.Stop(ctx)
}
