package relayer

import (
	"context"
	"log"

	"cosmossdk.io/depinject"
)

// relayMiner is the main struct that encapsulates the relayer's responsibilities (i.e. Relay Mining).
// It starts and stops the RelayerProxy and provide the served relays observable to the miner.
type relayMiner struct {
	relayerProxy           RelayerProxy
	miner                  Miner
	relayerSessionsManager RelayerSessionsManager
}

// NewRelayMiner creates a new Relayer instance with the given dependencies.
// It injects the dependencies into the Relayer instance and returns it.
func NewRelayMiner(ctx context.Context, deps depinject.Config) (*relayMiner, error) {
	rel := &relayMiner{}

	if err := depinject.Inject(
		deps,
		&rel.relayerProxy,
		&rel.miner,
		&rel.relayerSessionsManager,
	); err != nil {
		return nil, err
	}

	// Set up relay pipeline
	servedRelaysObs := rel.relayerProxy.ServedRelays()
	minedRelaysObs := rel.miner.MinedRelays(ctx, servedRelaysObs)
	rel.relayerSessionsManager.InsertRelays(minedRelaysObs)

	return rel, nil
}

// Start provides the miner with the served relays observable and starts the relayer proxy.
// This method is blocking while the relayer proxy is running and returns when Stop is called
// or when the relayer proxy fails to start.
func (rel *relayMiner) Start(ctx context.Context) error {
	// relayerSessionsManager.Start does not block.
	// Set up the session (proof/claim) lifecycle pipeline.
	log.Println("INFO: Starting relayer sessions manager...")
	rel.relayerSessionsManager.Start(ctx)

	// Start the flow of relays by starting relayer proxy.
	// This is a blocking call as it waits for the waitgroup in relayerProxy.Start()
	// that starts all the relay servers to be done.
	log.Println("INFO: Starting relayer proxy...")
	if err := rel.relayerProxy.Start(ctx); err != nil {
		return err
	}

	log.Println("INFO: Relayer proxy stopped; exiting")
	return nil
}

// Stop stops the relayer proxy which in turn stops all advertised relay servers
// and unsubscribes the miner from the served relays observable.
func (rel *relayMiner) Stop(ctx context.Context) error {
	return rel.relayerProxy.Stop(ctx)
}
