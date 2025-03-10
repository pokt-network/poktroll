package tx

import (
	"context"

	"cosmossdk.io/depinject"
	cometrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	cosmostx "github.com/cosmos/cosmos-sdk/client/tx"
	cosmoskeyring "github.com/cosmos/cosmos-sdk/crypto/keyring"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	authclient "github.com/cosmos/cosmos-sdk/x/auth/client"
	"google.golang.org/grpc"

	"github.com/pokt-network/poktroll/pkg/client"
	txtypes "github.com/pokt-network/poktroll/pkg/client/tx/types"
)

// maxGRPCMsgSize is the maximum message size the gRPC client can send and receive.
// The current value has been set arbitrarily to a large value after empirically
// observing multiple Proof messages bundled within a single transaction exceeding
// the default 4MB limit.
// TODO_MAINNET: Adjust the max message size to a more sensible value.
// DEV_NOTE: This value should adjusted in concert with the CometBFT's rpc
// max_body_bytes, mempool max_tx_bytes and max_txs_bytes.
const maxGRPCMsgSize = 100 * 1024 * 1024 // 100MB

var _ client.TxContext = (*cosmosTxContext)(nil)

// cosmosTxContext is an internal implementation of the client.TxContext interface.
// It provides methods related to transaction context within the Cosmos SDK.
type cosmosTxContext struct {
	// Holds cosmos-sdk client context.
	// (see: https://pkg.go.dev/github.com/cosmos/cosmos-sdk@v0.47.5/client#Context)
	clientCtx txtypes.Context
	// Holds the cosmos-sdk transaction factory.
	// (see: https://pkg.go.dev/github.com/cosmos/cosmos-sdk@v0.47.5/client/tx#Factory)
	txFactory cosmostx.Factory
}

// NewTxContext initializes a new cosmosTxContext with the given dependencies.
// It uses depinject to populate its members and returns a client.TxContext
// interface type.
//
// Required dependencies:
//   - cosmosclient.Context
//   - cosmostx.Factory
func NewTxContext(deps depinject.Config) (client.TxContext, error) {
	txCtx := cosmosTxContext{}

	if err := depinject.Inject(
		deps,
		&txCtx.clientCtx,
		&txCtx.txFactory,
	); err != nil {
		return nil, err
	}

	return txCtx, nil
}

// GetKeyring returns the cosmos-sdk client Keyring associated with the transaction factory.
func (txCtx cosmosTxContext) GetKeyring() cosmoskeyring.Keyring {
	return txCtx.txFactory.Keybase()
}

// SignTx signs the provided transaction using the given key name. It can operate in offline mode
// and can optionally overwrite any existing signatures.
// It is a proxy to the cosmos-sdk auth module client SignTx function.
// (see: https://pkg.go.dev/github.com/cosmos/cosmos-sdk@v0.47.5/x/auth/client)
func (txCtx cosmosTxContext) SignTx(
	signingKeyName string,
	txBuilder cosmosclient.TxBuilder,
	offline, overwriteSig bool,
) error {
	return authclient.SignTx(
		txCtx.txFactory,
		cosmosclient.Context(txCtx.clientCtx),
		signingKeyName,
		txBuilder,
		offline, overwriteSig,
	)
}

// NewTxBuilder returns a new transaction builder instance using the cosmos-sdk client transaction config.
func (txCtx cosmosTxContext) NewTxBuilder() cosmosclient.TxBuilder {
	return txCtx.clientCtx.TxConfig.NewTxBuilder()
}

// EncodeTx encodes the provided tx and returns its bytes representation.
func (txCtx cosmosTxContext) EncodeTx(txBuilder cosmosclient.TxBuilder) ([]byte, error) {
	return txCtx.clientCtx.TxConfig.TxEncoder()(txBuilder.GetTx())
}

// BroadcastTx broadcasts the given transaction to the network, blocking until the check-tx
// ABCI operation completes and returns a TxResponse of the transaction status at that point in time.
func (txCtx cosmosTxContext) BroadcastTx(txBytes []byte) (*cosmostypes.TxResponse, error) {
	clientCtx := cosmosclient.Context(txCtx.clientCtx)
	// BroadcastTxSync is used to capture any transaction error that occurs during
	// the check-tx ABCI operation, otherwise errors would not be returned.
	return clientCtx.BroadcastTxSync(txBytes)
}

// QueryTx queries the transaction based on its hash and optionally provides proof
// of the transaction. It returns the transaction query result.
func (txCtx cosmosTxContext) QueryTx(
	ctx context.Context,
	txHash []byte,
	prove bool,
) (*cometrpctypes.ResultTx, error) {
	return txCtx.clientCtx.Client.Tx(ctx, txHash, prove)
}

// GetClientCtx returns the cosmos-sdk client context associated with the transaction context.
func (txCtx cosmosTxContext) GetClientCtx() cosmosclient.Context {
	return cosmosclient.Context(txCtx.clientCtx)
}

// GetSimulatedTxGas calculates the gas for the given messages using the simulation mode.
func (txCtx cosmosTxContext) GetSimulatedTxGas(
	ctx context.Context,
	signingKeyName string,
	msgs ...cosmostypes.Msg,
) (uint64, error) {
	clientCtx := cosmosclient.Context(txCtx.clientCtx)
	keyRecord, err := txCtx.GetKeyring().Key(signingKeyName)
	if err != nil {
		return 0, err
	}

	accAddress, err := keyRecord.GetAddress()
	if err != nil {
		return 0, err
	}

	accountRetriever := txCtx.clientCtx.AccountRetriever
	_, seq, err := accountRetriever.GetAccountNumberSequence(clientCtx, accAddress)
	if err != nil {
		return 0, err
	}

	txf := txCtx.txFactory.
		WithSimulateAndExecute(true).
		WithFromName(signingKeyName).
		WithSequence(seq)

	txBytes, err := txf.BuildSimTx(msgs...)
	if err != nil {
		return 0, err
	}

	txSvcClient := tx.NewServiceClient(clientCtx)

	simRequest := &tx.SimulateRequest{TxBytes: txBytes}
	// Set the maximum message size for the gRPC client to allow large transactions
	// (e.g. transactions with multiple proof messages) to be simulated.
	gRPCOpts := []grpc.CallOption{
		grpc.MaxCallSendMsgSize(maxGRPCMsgSize),
		grpc.MaxCallRecvMsgSize(maxGRPCMsgSize),
	}
	simRes, err := txSvcClient.Simulate(context.Background(), simRequest, gRPCOpts...)
	if err != nil {
		return 0, err
	}

	return uint64(txf.GasAdjustment() * float64(simRes.GasInfo.GasUsed)), nil
}
