package relayer

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// ServeForward exposes a forward HTTP server for administrators to send request to
// specific service.
func (rel *relayMiner) ServeForward(ctx context.Context, network, addr, token string) error {
	ln, err := net.Listen(network, addr)
	if err != nil {
		return fmt.Errorf("net listen: %w", err)
	}

	muxRouter := chi.NewRouter()
	muxRouter.HandleFunc("/services/{service_id}/forward", rel.newForwardHandlerFn(ctx, token))

	go func() {
		if err := http.Serve(ln, muxRouter); err != nil {
			rel.logger.Error().Err(err).
				Msg("unexpected error occurred while serving forward server")
			return
		}
	}()

	return nil
}

func (rel *relayMiner) newForwardHandlerFn(ctx context.Context, token string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqToken := r.Header.Get("token")
		if reqToken != token {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		serviceID := chi.URLParam(r, "service_id")
		if serviceID == "" {
			rel.logger.Error().Msg("service id not found in URL while forwarding request")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		rel.logger.Debug().Str("service_id", serviceID).
			Msg("forwarding request to supplier...")

		if err := rel.relayerProxy.Forward(ctx, serviceID, w, r); err != nil {
			rel.logger.Error().Err(err).
				Msg("unable to forward request")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})
}
