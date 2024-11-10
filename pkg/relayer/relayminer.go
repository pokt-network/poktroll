package relayer

import (
	"context"
	"net"
	"net/http"
	"net/http/pprof"

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
	// TODO_TECHDEBT: Listen for onchain and local configuration changes, stop
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

// Starts a pprof server on the given address.
func (rel *relayMiner) ServePprof(ctx context.Context, addr string) error {
	pprofMux := http.NewServeMux()
	pprofMux.HandleFunc("/debug/pprof/", pprof.Index)
	pprofMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	pprofMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	pprofMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	pprofMux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	server := &http.Server{
		Addr:    addr,
		Handler: pprofMux,
	}
	// If no error, start the server in a new goroutine
	go func() {
		rel.logger.Info().Str("endpoint", addr).Msg("starting a pprof endpoint")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			rel.logger.Error().Str("endpoint", addr).Msg("unable to start a pprof endpoint")
		}
	}()

	go func() {
		<-ctx.Done()
		rel.logger.Info().Str("endpoint", addr).Msg("stopping a pprof endpoint")
		_ = server.Shutdown(ctx)
	}()

	return nil
}

// ServePing exposes ping HTTP server to check the reachability between the
// relay miner and its dependencies (Ex: relay server and their respective
// backend URLs).
func (rel *relayMiner) ServePing(ctx context.Context, ln net.Listener) {
	// Start a long-lived goroutine that starts an HTTP server responding to
	// ping requests. A single ping request on the relay server broadcasts a
	// ping to all backing services/data nodes.
	go func() {
		if err := http.Serve(ln, rel.newPinghandlerFn(ctx, ln)); err != nil {
			rel.logger.Error().Err(err).Msg("unable to serve ping server")
		}
	}()

	return
}

func (rel *relayMiner) newPinghandlerFn(ctx context.Context, ln net.Listener) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		rel.logger.Debug().Msg("pinging relay servers...")

		if err := rel.relayerProxy.PingAll(ctx); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})
}
