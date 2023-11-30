package tx

import (
	"context"

	"cosmossdk.io/depinject"
	cometrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	cosmostx "github.com/cosmos/cosmos-sdk/client/tx"
	cosmoskeyring "github.com/cosmos/cosmos-sdk/crypto/keyring"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authclient "github.com/cosmos/cosmos-sdk/x/auth/client"

	"github.com/pokt-network/poktroll/pkg/client"
	txtypes "github.com/pokt-network/poktroll/pkg/client/tx/types"
)

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
	return clientCtx.BroadcastTxAsync(txBytes)
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
