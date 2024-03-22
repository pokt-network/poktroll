package relayer

import (
	"context"
	"net"
	"net/http"

	"cosmossdk.io/depinject"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/pokt-network/poktroll/pkg/polylog"
)

// relayMiner is the main struct that encapsulates the relayer's responsibilities (i.e. Relay Mining).
// It starts and stops the RelayerProxy and provide the served relays observable to the miner.
type relayMiner struct {
	logger                 polylog.Logger
	relayerProxy           RelayerProxy
	miner                  Miner
	relayerSessionsManager RelayerSessionsManager
}

// NewRelayMiner creates a new Relayer instance with the given dependencies.
// It injects the dependencies into the Relayer instance and returns it.
//
// Required dependencies:
//   - RelayerProxy
//   - Miner
//   - RelayerSessionsManager
func NewRelayMiner(ctx context.Context, deps depinject.Config) (*relayMiner, error) {
	rel := &relayMiner{
		logger: polylog.Ctx(ctx),
	}

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
	rel.logger.Info().Msg("starting relayer sessions manager")
	rel.relayerSessionsManager.Start(ctx)

	// Start the flow of relays by starting relayer proxy.
	// This is a blocking call as it waits for the waitgroup in relayerProxy.Start()
	// that starts all the relay servers to be done.
	rel.logger.Info().Msg("starting relayer proxy")
	// TODO_TECHDEBT: Listen for on-chain and local configuration changes, stop
	// the relayerProxy if they do not match, then wait until they match again
	// before starting the relayerProxy with the new config.
	// Session manager should continue to run during this time, submitting
	// any relays that were already mined in previous sessions.
	// Link to more context:
	// https://github.com/pokt-network/poktroll/assets/231488/297a3889-85a4-4c13-a852-f2afc10b2be3
	if err := rel.relayerProxy.Start(ctx); err != nil {
		return err
	}

	rel.logger.Info().Msg("relayer proxy stopped; exiting")
	return nil
}

// Stop stops the relayer proxy which in turn stops all advertised relay servers
// and unsubscribes the miner from the served relays observable.
func (rel *relayMiner) Stop(ctx context.Context) error {
	rel.relayerSessionsManager.Stop()
	return rel.relayerProxy.Stop(ctx)
}

// Starts a metrics server on the given address.
func (rel *relayMiner) ServeMetrics(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		rel.logger.Error().Err(err).Msg("failed to listen on address for metrics")
		return err
	}

	// If no error, start the server in a new goroutine
	go func() {
		rel.logger.Info().Str("endpoint", addr).Msg("serving metrics")
		if err := http.Serve(ln, promhttp.Handler()); err != nil {
			rel.logger.Error().Err(err).Msg("metrics server failed")
			return
		}
	}()

	return nil
}
