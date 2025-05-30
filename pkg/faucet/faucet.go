package faucet

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/go-chi/chi/v5"

	"github.com/pokt-network/poktroll/cmd/logger"
	"github.com/pokt-network/poktroll/pkg/client/query"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

const denomPathFmt = "/%s/"

// Server is an HTTP server that responds to funding requests, parameterized
// by address and denomination, by broadcasting bank send transactions onchain
// using a dedicated "faucet" account.
type Server struct {
	config  *Config
	handler *chi.Mux
}

// NewFaucetServer returns a new Server instance, configured according to the provided options.
func NewFaucetServer(ctx context.Context, opts ...FaucetOptionFn) (*Server, error) {
	faucet := &Server{
		config:  new(Config),
		handler: chi.NewRouter(),
	}

	for _, opt := range opts {
		opt(faucet)
	}

	//for _, sendCoin := range faucet.config.GetSupportedSendCoins() {
	handleDenomRequest := faucet.newHandleDenomRequest(ctx)
	// TODO_IN_THIS_COMMIT: promote to a const.
	// TODO_IN_THIS_COMMIT: extract param names to own consts.
	denomPathRouteTemplate := "/{denom}/{recipient_address}"
	faucet.handler.Post(denomPathRouteTemplate, handleDenomRequest)
	//}

	return faucet, nil
}

// Serve starts the HTTP server that responds to funding requests.
// It is a blocking operation that will not return until the given context is canceled.
func (srv *Server) Serve(ctx context.Context) error {
	httpServer := &http.Server{
		Addr:    srv.config.ListenAddress,
		Handler: srv.handler,
	}

	go func(httpServer *http.Server) {
		<-ctx.Done()
		_ = httpServer.Shutdown(context.Background())
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
func (srv *Server) newHandleDenomRequest(ctx context.Context) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		// TDOO_IN_THIS_COMMIT: Promote to a const.
		denom := chi.URLParam(req, "denom")

		// TODO_IN_THIS_COMMIT: update comments...
		// Add routes for each supported denomination:
		// - Denomination as configured (e.g. "uPOKT")
		// - Uppercase denomination (e.g. "UPOKT")
		// - Lowercase denomination (e.g. "upokt")
		sendCoin := new(cosmostypes.Coin)
		for _, supportedSendCoin := range srv.config.GetSupportedSendCoins() {
			switch supportedSendCoin.Denom {
			case strings.ToUpper(denom):
				fallthrough
			case strings.ToLower(denom):
				fallthrough
			case denom:
				*sendCoin = supportedSendCoin
			}

			if sendCoin != nil {
				break
			}
		}
		if sendCoin == nil {
			// TODO_IN_THIS_COMMIT:
			// - Return a 404 error
			// - include the unsupported denom in the body
		}

		logger := logger.Logger.With("denom", sendCoin.Denom)

		// TDOO_IN_THIS_COMMIT: Promote to a const.
		recipientAddressStr := chi.URLParam(req, "recipient_address")
		recipientAddress, err := cosmostypes.AccAddressFromBech32(recipientAddressStr)
		if err != nil {
			respondBadRequest(logger, res, err)
			return
		}

		logger = logger.With("recipient_address", recipientAddress)

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
		txResponse, sendErr := srv.SendDenom(ctx, logger, denom, recipientAddress)
		if sendErr != nil {
			respondBadRequest(logger, res, sendErr)
			return
		}

		// If CheckTx fails, return a 400 Bad Request response with the tx log.
		if txResponse.Code != 0 {
			respondBadRequest(logger, res, errors.New(txResponse.RawLog))
			return
		}

		// Send accepted response with tx hash (202).
		respondAccepted(logger, res, txResponse.TxHash)
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
// It ONLY ensures that the send TX passed the CheckTx ABCI method (i.e. made it into the mempool).
// It DOES NOT wait for the TX to be committed AND there is a possibility that the TX will fail.
func (srv *Server) SendDenom(
	ctx context.Context,
	logger polylog.Logger,
	denom string,
	recipientAddress cosmostypes.AccAddress,
) (*cosmostypes.TxResponse, error) {
	isDenomSupported, denomCoin := srv.config.GetSupportedSendCoins().Find(denom)
	if !isDenomSupported {
		return nil, fmt.Errorf("denom %q not supported", denom)
	}

	sendMsg := banktypes.NewMsgSend(
		srv.config.signingAddress,
		recipientAddress,
		cosmostypes.NewCoins(denomCoin),
	)

	txResponse, eitherErr := srv.config.txClient.SignAndBroadcast(ctx, sendMsg)
	err, errCh := eitherErr.SyncOrAsyncError()
	if err != nil {
		return nil, err
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

	return txResponse, nil
}

// respondAccepted sends a 202 Accepted response to the given http.ResponseWriter.
func respondAccepted(logger polylog.Logger, res http.ResponseWriter, msg string) {
	// Send a accepted response (202).
	res.WriteHeader(http.StatusAccepted)
	if _, err := res.Write([]byte(msg)); err != nil {
		logger.Error().Err(err).Send()
	}
}

// respondBadRequest sends a 400 Bad Request response to the given http.ResponseWriter and logs the given error.
func respondBadRequest(logger polylog.Logger, res http.ResponseWriter, err error) {
	logger.Error().Err(err).Send()

	// Send a bad request response (400).
	res.WriteHeader(http.StatusBadRequest)
	if _, err = res.Write([]byte(err.Error())); err != nil {
		logger.Error().Err(err).Send()
	}
}

// respondInternalError sends a 500 Internal Server Error response to the given http.ResponseWriter and logs the given error.
func respondInternalError(logger polylog.Logger, res http.ResponseWriter, err error) {
	logger.Error().Err(err).Send()

	res.WriteHeader(http.StatusInternalServerError)
	if _, err = res.Write([]byte(err.Error())); err != nil {
		logger.Error().Err(err).Send()
	}
}

// respondNotModified sends a 304 Not Modified response to the given http.ResponseWriter and logs the given recipientAddress.
func respondNotModified(logger polylog.Logger, res http.ResponseWriter, recipientAddress string) {
	res.WriteHeader(http.StatusNotModified)
	if _, err := fmt.Fprintf(res, "address %s already exists onchain", recipientAddress); err != nil {
		logger.Error().Err(err).Send()
	}
}
