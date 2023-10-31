package tx

import (
	"context"

	"cosmossdk.io/depinject"
	cometrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	cosmostx "github.com/cosmos/cosmos-sdk/client/tx"
	cosmoskeyring "github.com/cosmos/cosmos-sdk/crypto/keyring"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authClient "github.com/cosmos/cosmos-sdk/x/auth/client"

	"pocket/pkg/client"
)

var _ client.TxContext = (*cosmosTxContext)(nil)

type cosmosTxContext struct {
	clientCtx cosmosclient.Context
	txFactory cosmostx.Factory
}

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

func (txCtx cosmosTxContext) GetKeyring() cosmoskeyring.Keyring {
	return txCtx.txFactory.Keybase()
}

func (txCtx cosmosTxContext) SignTx(
	signingKeyName string,
	txBuilder cosmosclient.TxBuilder,
	offline, overwriteSig bool,
) error {
	return authClient.SignTx(
		txCtx.txFactory,
		txCtx.clientCtx,
		signingKeyName,
		txBuilder,
		offline, overwriteSig,
	)
}

func (txCtx cosmosTxContext) NewTxBuilder() cosmosclient.TxBuilder {
	return txCtx.clientCtx.TxConfig.NewTxBuilder()
}

func (txCtx cosmosTxContext) EncodeTx(txBuilder cosmosclient.TxBuilder) ([]byte, error) {
	return txCtx.clientCtx.TxConfig.TxEncoder()(txBuilder.GetTx())
}

func (txCtx cosmosTxContext) BroadcastTxSync(txBytes []byte) (*cosmostypes.TxResponse, error) {
	return txCtx.clientCtx.BroadcastTxSync(txBytes)
}

func (txCtx cosmosTxContext) QueryTx(
	ctx context.Context,
	txHash []byte,
	prove bool,
) (*cometrpctypes.ResultTx, error) {
	return txCtx.clientCtx.Client.Tx(ctx, txHash, prove)
}
