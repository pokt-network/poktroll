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

const (
	denomRouteDenomParamName            = "denom"
	denomRouteRecipientAddressParamName = "recipient_address"
)

// denomRouteTemplate defines the HTTP route for denomination funding requests.
// Format: /{denom}/{recipient_address}
var denomRouteTemplate = fmt.Sprintf("/{%s}/{%s}", denomRouteDenomParamName, denomRouteRecipientAddressParamName)

// Server handles HTTP funding requests by broadcasting onchain bank send transactions
// using a dedicated faucet account.
type Server struct {
	config  *FaucetConfig
	handler *chi.Mux
}

// NewFaucetServer constructs a new Server using the provided options.
func NewFaucetServer(ctx context.Context, opts ...FaucetOptionFn) (*Server, error) {
	faucet := &Server{
		config:  new(FaucetConfig),
		handler: chi.NewRouter(),
	}

	for _, opt := range opts {
		opt(faucet)
	}

	handleDenomRequest := faucet.newHandleDenomPOSTRequest(ctx)
	faucet.handler.Post(denomRouteTemplate, handleDenomRequest)

	return faucet, nil
}

// Serve starts the HTTP server and blocks until the context is canceled.
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

// GetSigningAddress returns the faucet's current signing address as a string.
func (srv *Server) GetSigningAddress() string {
	return srv.config.signingAddress.String()
}

// GetBalances queries the onchain balances for the specified address.
func (srv *Server) GetBalances(ctx context.Context, address string) (cosmostypes.Coins, error) {
	balancesRes, err := srv.config.bankQueryClient.AllBalances(ctx, &banktypes.QueryAllBalancesRequest{
		Address: address,
	})
	if err != nil {
		return nil, err
	}

	return balancesRes.Balances, nil
}

// newHandleDenomPOSTRequest returns an HTTP handler for POST funding requests for a denomination.
// Context is canceled when the server shuts down.
func (srv *Server) newHandleDenomPOSTRequest(ctx context.Context) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		denom := chi.URLParam(req, denomRouteDenomParamName)

		// Respond to the following routes for each supported denomination:
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
			respondNotFound(logger.Logger, res, fmt.Errorf("unsupported denom %q", denom))
		}

		logger := logger.Logger.With("denom", sendCoin.Denom)

		recipientAddressStr := chi.URLParam(req, denomRouteRecipientAddressParamName)
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

// shouldSendToRecipient determines if the faucet should send tokens to recipientAddress.
// - If CreateAccountsOnly is false: always returns true.
// - If CreateAccountsOnly is true: returns true only if the address does not exist onchain (no balances).
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

// SendDenom sends tokens of the specified denom to recipientAddress.
// - Amount is determined by SupportedSendCoins config.
// - Only checks that the TX passed CheckTx (entered mempool); does not wait for commit.
// - TX may still fail after being accepted into the mempool.
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
	logger.Info().Msg("transaction sent")

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
				logger.Info().Msg("transaction succeeded")
			}
			return
		}
	}(ctx, txResponse, errCh)

	return txResponse, nil
}

// respondAccepted writes a 202 Accepted response with the provided message.
func respondAccepted(logger polylog.Logger, res http.ResponseWriter, msg string) {
	// Send a accepted response (202).
	res.WriteHeader(http.StatusAccepted)
	if _, err := res.Write([]byte(msg)); err != nil {
		logger.Error().Err(err).Send()
	}
}

// respondBadRequest writes a 400 Bad Request response and logs the error.
func respondBadRequest(logger polylog.Logger, res http.ResponseWriter, err error) {
	logger.Error().Err(err).Send()

	// Send a bad request response (400).
	res.WriteHeader(http.StatusBadRequest)
	if _, err = res.Write([]byte(err.Error())); err != nil {
		logger.Error().Err(err).Send()
	}
}

// respondInternalError writes a 500 Internal Server Error response and logs the error.
func respondInternalError(logger polylog.Logger, res http.ResponseWriter, err error) {
	logger.Error().Err(err).Send()

	res.WriteHeader(http.StatusInternalServerError)
	if _, err = res.Write([]byte(err.Error())); err != nil {
		logger.Error().Err(err).Send()
	}
}

// respondNotModified writes a 304 Not Modified response and logs the recipient address.
func respondNotModified(logger polylog.Logger, res http.ResponseWriter, recipientAddress string) {
	res.WriteHeader(http.StatusNotModified)
	if _, err := fmt.Fprintf(res, "address %s already exists onchain", recipientAddress); err != nil {
		logger.Error().Err(err).Send()
	}
}

// respondNotFound writes a 404 Not Found response and logs the error.
func respondNotFound(logger polylog.Logger, res http.ResponseWriter, err error) {
	logger.Error().Err(err).Send()
	res.WriteHeader(http.StatusNotFound)
	if _, err := fmt.Fprintf(res, "not found"); err != nil {
		logger.Error().Err(err).Send()
	}
}
