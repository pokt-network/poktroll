package relayer

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/pprof"
	"net/url"

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

// ServePing starts an HTTP server that:
// - Checks connectivity between relay miner and dependencies
// - Tests reachability of relay servers and their backend URLs
func (rel *relayMiner) ServePing(ctx context.Context, network, addr string) error {
	ln, err := net.Listen(network, addr)
	if err != nil {
		return err
	}

	// Starts a go routine that:
	// - Create a long-running HTTP server
	// - Handles ping requests by broadcasting health checks to all backing services
	// - Tests connectivity to all configured data nodes
	go func() {
		if err := http.Serve(ln, rel.newPingHandlerFn(ctx)); err != nil && !errors.Is(err, http.ErrServerClosed) {
			rel.logger.Error().Err(err).Msg("ping server unexpectedly closed")
		}
	}()

	go func() {
		<-ctx.Done() // A message a receive when we stop the relayminer.
		rel.logger.Info().Str("endpoint", addr).Msg("stopping ping server")
		_ = ln.Close()
	}()

	return nil
}

func (rel *relayMiner) newPingHandlerFn(ctx context.Context) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		rel.logger.Debug().Msg("pinging relay servers...")

		if err := rel.relayerProxy.PingAll(ctx); err != nil {
			var urlError *url.Error
			if errors.As(err, &urlError) && urlError.Temporary() {
				w.WriteHeader(http.StatusGatewayTimeout)
			} else {
				w.WriteHeader(http.StatusBadGateway)
			}
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})
}
