package faucet

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/pokt-network/poktroll/cmd/logger"
	"github.com/pokt-network/poktroll/pkg/client/query"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

const denomPathFmt = "/%s/"

// Server is an HTTP server that responds to funding requests, parameterized
// by address and denomination, by broadcasting bank send transactions onchain
// using a dedicated "faucet" account.
type Server struct {
	config   *Config
	listener net.Listener
	httpMux  *http.ServeMux
}

// NewFaucetServer returns a new Server instance, configured according to the provided options.
func NewFaucetServer(ctx context.Context, opts ...FaucetOptionFn) (*Server, error) {
	faucet := &Server{
		config:  new(Config),
		httpMux: http.NewServeMux(),
	}

	for _, opt := range opts {
		opt(faucet)
	}

	// Add routes for each supported denomination:
	// - Denomination as configured (e.g. "uPOKT")
	// - Uppercase denomination (e.g. "UPOKT")
	// - Lowercase denomination (e.g. "upokt")
	for _, sendCoin := range faucet.config.GetSupportedSendCoins() {
		denomPath := fmt.Sprintf(denomPathFmt, sendCoin.Denom)
		denomPathUpper := fmt.Sprintf(denomPathFmt, strings.ToUpper(sendCoin.Denom))
		denomPathLower := fmt.Sprintf(denomPathFmt, strings.ToLower(sendCoin.Denom))

		handleDenomRequest := faucet.newHandleDenomRequest(ctx, sendCoin.Denom)
		faucet.httpMux.HandleFunc(denomPath, handleDenomRequest)

		// Don't add duplicate routes.
		if denomPathUpper != denomPath {
			faucet.httpMux.HandleFunc(denomPathUpper, handleDenomRequest)
		}
		if denomPathLower != denomPath {
			faucet.httpMux.HandleFunc(denomPathLower, handleDenomRequest)
		}
	}

	return faucet, nil
}

// Serve starts the HTTP server that responds to funding requests.
// It is a blocking operation that will not return until the given context is canceled.
func (srv *Server) Serve(ctx context.Context) error {
	httpServer := &http.Server{
		Addr:    srv.config.ListenAddress,
		Handler: srv.httpMux,
	}

	go func(httpServer *http.Server) {
		select {
		case <-ctx.Done():
			_ = httpServer.Shutdown(context.Background())
		}
	}(httpServer)

	err := httpServer.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// GetSigningAddress returns the address of the faucet's configured signing key.
func (srv *Server) GetSigningAddress() string {
	return srv.config.signingAddress.String()
}

// GetBalances queries for the onchain balances of the given address.
func (srv *Server) GetBalances(ctx context.Context, address string) (cosmostypes.Coins, error) {
	balancesRes, err := srv.config.bankQueryClient.AllBalances(ctx, &banktypes.QueryAllBalancesRequest{
		Address: address,
	})
	if err != nil {
		return nil, err
	}

	return balancesRes.Balances, nil
}

// newHandleDenomRequest is a handler factory function that returns a new HTTP handler
// which responds to funding requests for the given denomination.
func (srv *Server) newHandleDenomRequest(ctx context.Context, denom string) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		// Extract the recipient address from the URL path.
		denomPath := fmt.Sprintf(denomPathFmt, denom)
		recipientAddressStr := req.URL.Path[len(denomPath):]
		recipientAddress, err := cosmostypes.AccAddressFromBech32(recipientAddressStr)
		if err != nil {
			logger.Logger.Error().Err(err).Send()
			res.WriteHeader(http.StatusBadRequest)
			if _, err = res.Write([]byte(err.Error())); err != nil {
				logger.Logger.Error().Err(err).Send()
			}
			return
		}

		logger := logger.Logger.With("recipient_address", recipientAddress)

		// Check if the recipient address already exists onchain.
		shouldSend, queryErr := srv.shouldSendToRecipient(req.Context(), recipientAddress)
		if queryErr != nil {
			respondInternalError(logger, res, queryErr)
			return
		}

		// If the create_account_only is true, address already exists onchain.
		// Return a "not modified" response.
		if !shouldSend {
			respondNotModified(logger, res, recipientAddress.String())
			return
		}

		// If the address doesn't exist onchain, send it tokens.
		if err = srv.SendDenom(ctx, logger, denom, recipientAddress); err != nil {
			// TODO_IN_THIS_COMMIT: not all error cases are 400 Bad Request...
			respondBadRequest(res, err)
			return
		}

		// Send empty success response (200).
		if _, err = res.Write([]byte{}); err != nil {
			logger.Error().Err(err).Send()
			return
		}
	}
}

// shouldSendToRecipient indicates whether the faucet should send tokens to the given recipient address.
// When create_accounts_only is false:
// - ALWAYS returns true
// When create_accounts_only is true:
// - ONLY return true IF the recipient address does NOT already exist onchain (i.e. has no balances)
func (srv *Server) shouldSendToRecipient(ctx context.Context, recipientAddress cosmostypes.AccAddress) (bool, error) {
	if srv.config.CreateAccountsOnly {
		_, err := srv.GetBalances(ctx, recipientAddress.String())
		switch {
		case errors.Is(err, query.ErrQueryBalanceNotFound):
			return true, nil
		case err != nil:
			return false, err
		default:
			// The account has a balance; therefore, it already exists onchain.
			return false, nil
		}
	}

	// Send by default.
	return true, nil
}

// SendDenom sends tokens of the given denomination to the given recipient address.
// The amount of tokens sent is determined by the faucet's supported_send_coins configuration.
func (srv *Server) SendDenom(
	ctx context.Context,
	logger polylog.Logger,
	denom string,
	recipientAddress cosmostypes.AccAddress,
) error {
	isDenomSupported, denomCoin := srv.config.GetSupportedSendCoins().Find(denom)
	if !isDenomSupported {
		return fmt.Errorf("denom %q not supported", denom)
	}

	sendMsg := banktypes.NewMsgSend(
		srv.config.signingAddress,
		recipientAddress,
		cosmostypes.NewCoins(denomCoin),
	)

	txResponse, eitherErr := srv.config.txClient.SignAndBroadcast(ctx, sendMsg)
	err, errCh := eitherErr.SyncOrAsyncError()
	if err != nil {
		return err
	}

	logger = logger.With("tx_hash", txResponse.TxHash)
	logger.Debug().Msg("transaction sent")

	go func(
		ctx context.Context,
		txResponse *cosmostypes.TxResponse,
		errCh <-chan error,
	) {
		select {
		case <-ctx.Done():
			return
		case asyncErr := <-errCh:
			if asyncErr != nil {
				logger.Error().Err(asyncErr).Msg("transaction failed")
			} else {
				logger.Debug().Msg("transaction succeeded")
			}
			return
		}
	}(ctx, txResponse, errCh)

	return nil
}

// TODO_IN_THIS_COMMIT: godoc & move...
func respondBadRequest(res http.ResponseWriter, err error) {
	logger.Logger.Error().Err(err).Send()

	// Send a bad request response (400).
	res.WriteHeader(http.StatusBadRequest)
	if _, err = res.Write([]byte(err.Error())); err != nil {
		logger.Logger.Error().Err(err).Send()
	}
}

// TODO_IN_THIS_COMMIT: godoc & move...
func respondInternalError(logger polylog.Logger, res http.ResponseWriter, err error) {
	logger.Error().Err(err).Send()

	res.WriteHeader(http.StatusInternalServerError)
	if _, err = res.Write([]byte(err.Error())); err != nil {
		logger.Error().Err(err).Send()
	}
}

// TODO_IN_THIS_COMMIT: godoc & move...
func respondNotModified(logger polylog.Logger, res http.ResponseWriter, recipientAddress string) {
	res.WriteHeader(http.StatusNotModified)
	if _, err := fmt.Fprintf(res, "address %s already exists onchain", recipientAddress); err != nil {
		logger.Error().Err(err).Str("recipient_address", recipientAddress).Send()
	}
}
