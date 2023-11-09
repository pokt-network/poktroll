package relayer

import (
	"context"

	"cosmossdk.io/depinject"
)

type RelayerOption func(*Relayer)

// Relayer is the main struct that encapsulates the relayer's responsibilities.
// It starts and stops the RelayerProxy and provide the served relays observable to them miner.
type Relayer struct {
	relayerProxy RelayerProxy
	miner        Miner
}

// NewRelayer creates a new Relayer instance with the given dependencies.
// It injects the dependencies into the Relayer instance and returns it.
func NewRelayer(
	deps depinject.Config,
	opts ...RelayerOption,
) (*Relayer, error) {
	rel := &Relayer{}

	if err := depinject.Inject(deps, &rel.relayerProxy, &rel.miner); err != nil {
		return nil, err
	}

	for _, opt := range opts {
		opt(rel)
	}

	return rel, nil
}

// Start provides the miner with the served relays observable and starts the relayer proxy.
// This method is blocking while the relayer proxy is running and returns when Stop is called
// or when the relayer proxy fails to start.
func (rel *Relayer) Start(ctx context.Context) error {
	rel.miner.MineRelays(ctx, rel.relayerProxy.ServedRelays())
	return rel.relayerProxy.Start(ctx)
}

// Stop stops the relayer proxy which in turn stops all advertised relay servers
// and unsubscribes the miner from the served relays observable.
func (rel *Relayer) Stop(ctx context.Context) error {
	return rel.relayerProxy.Stop(ctx)
}
