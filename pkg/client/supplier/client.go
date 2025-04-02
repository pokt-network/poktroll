package supplier

import (
	"context"
	"sync"

	"cosmossdk.io/depinject"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/keyring"
	"github.com/pokt-network/poktroll/pkg/polylog"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
)

var _ client.SupplierClient = (*supplierClient)(nil)

// supplierClient
type supplierClient struct {
	// signingKeyName is the name of the operator key in the keyring that will be
	// used to sign transactions.
	signingKeyName string
	// signingKeyAddr is the account address of the operator key in the keyring.
	signingKeyAddr cosmostypes.AccAddress

	// pendingTxMu is used to prevent concurrent txs with the same sequence number.
	pendingTxMu sync.Mutex

	txClient client.TxClient
	txCtx    client.TxContext
}

// NewSupplierClient constructs a new SupplierClient with the given dependencies
// and options. If a signingKeyName is not configured, an error will be returned.
//
// Required dependencies:
//   - client.TxClient
//   - client.TxContext
//
// Available options:
//   - WithSigningKeyName
func NewSupplierClient(
	deps depinject.Config,
	opts ...client.SupplierClientOption,
) (*supplierClient, error) {
	sClient := &supplierClient{}

	if err := depinject.Inject(
		deps,
		&sClient.txClient,
		&sClient.txCtx,
	); err != nil {
		return nil, err
	}

	for _, opt := range opts {
		opt(sClient)
	}

	if err := sClient.validateConfigAndSetDefaults(); err != nil {
		return nil, err
	}

	return sClient, nil
}

// SubmitProofs constructs submit proof messages into a single transaction
// then signs and broadcasts it to the network via #txClient. It blocks until
// the transaction is included in a block or times out.
func (sClient *supplierClient) SubmitProofs(
	ctx context.Context,
	timeoutHeight int64,
	proofMsgs ...client.MsgSubmitProof,
) error {
	sClient.pendingTxMu.Lock()
	defer sClient.pendingTxMu.Unlock()
	logger := polylog.Ctx(ctx)

	msgs := make([]cosmostypes.Msg, 0, len(proofMsgs))
	for _, p := range proofMsgs {
		msgs = append(msgs, p)
	}

	// TODO(@bryanchriswhite): reconcile splitting of supplier & proof modules
	//  with offchain pkgs/nomenclature.
	eitherErr := sClient.txClient.SignAndBroadcast(ctx, timeoutHeight, msgs...)
	err, errCh := eitherErr.SyncOrAsyncError()
	if err != nil {
		return err
	}

	for _, p := range proofMsgs {
		// Type casting does not need to be checked here since the concrete type is
		// guaranteed to implement the interface which is just an identity for the
		// concrete type.
		proofMsg, _ := p.(*prooftypes.MsgSubmitProof)
		sessionHeader := proofMsg.SessionHeader
		// TODO_IMPROVE: log details related to what & how much is being proven
		logger.Info().
			Fields(map[string]any{
				"supplier_operator_addr": proofMsg.SupplierOperatorAddress,
				"app_addr":               sessionHeader.ApplicationAddress,
				"session_id":             sessionHeader.SessionId,
				"service_id":             sessionHeader.ServiceId,
			}).
			Msg("submitted a new proof")
	}

	return <-errCh
}

// CreateClaim constructs create claim messages into a single transaction
// then signs and broadcasts it to the network via #txClient. It blocks until
// the transaction is included in a block or times out.
func (sClient *supplierClient) CreateClaims(
	ctx context.Context,
	timeoutHeight int64,
	claimMsgs ...client.MsgCreateClaim,
) error {
	// Prevent concurrent txs with the same sequence number.
	sClient.pendingTxMu.Lock()
	defer sClient.pendingTxMu.Unlock()

	logger := polylog.Ctx(ctx)

	msgs := make([]cosmostypes.Msg, 0, len(claimMsgs))
	for _, c := range claimMsgs {
		msgs = append(msgs, c)
	}

	// TODO(@bryanchriswhite): reconcile splitting of supplier & proof modules
	//  with offchain pkgs/nomenclature.
	eitherErr := sClient.txClient.SignAndBroadcast(ctx, timeoutHeight, msgs...)
	err, errCh := eitherErr.SyncOrAsyncError()
	if err != nil {
		return err
	}

	for _, c := range claimMsgs {
		// Type casting does not need to be checked here since the concrete type is
		// guaranteed to implement the interface which is just an identity for the
		// concrete type.
		claimMsg, _ := c.(*prooftypes.MsgCreateClaim)
		sessionHeader := claimMsg.SessionHeader
		// TODO_IMPROVE: log details related to how much is claimed
		logger.Info().
			Fields(map[string]any{
				"supplier_operator_addr": claimMsg.SupplierOperatorAddress,
				"app_addr":               sessionHeader.ApplicationAddress,
				"session_id":             sessionHeader.SessionId,
				"service_id":             sessionHeader.ServiceId,
			}).
			Msg("created a new claim")
	}

	return <-errCh
}

// Address returns an address of the supplier client.
func (sClient *supplierClient) OperatorAddress() *cosmostypes.AccAddress {
	return &sClient.signingKeyAddr
}

// validateConfigAndSetDefaults attempts to get the address from the keyring
// corresponding to the key whose name matches the configured signingKeyName.
// If signingKeyName is empty or the keyring does not contain the corresponding
// key, an error is returned.
func (sClient *supplierClient) validateConfigAndSetDefaults() error {
	signingAddr, err := keyring.KeyNameToAddr(
		sClient.signingKeyName,
		sClient.txCtx.GetKeyring(),
	)
	if err != nil {
		return err
	}

	sClient.signingKeyAddr = signingAddr

	return nil
}
