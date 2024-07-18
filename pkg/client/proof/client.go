package proof

import (
	"context"
	"sync"

	"cosmossdk.io/depinject"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/keyring"
	"github.com/pokt-network/poktroll/pkg/polylog"
	prooftypes "github.com/pokt-network/poktroll/proto/types/proof"
)

var _ client.ProofClient = (*proofClient)(nil)

// proofClient
type proofClient struct {
	signingKeyName string
	signingKeyAddr cosmostypes.AccAddress

	// pendingTxMu is used to prevent concurrent txs with the same sequence number.
	pendingTxMu sync.Mutex

	txClient client.TxClient
	txCtx    client.TxContext
}

// NewProofClient constructs a new ProofClient with the given dependencies
// and options. If a signingKeyName is not configured, an error will be returned.
//
// Required dependencies:
//   - client.TxClient
//   - client.TxContext
//
// Available options:
//   - WithSigningKeyName
func NewProofClient(
	deps depinject.Config,
	opts ...client.SupplierClientOption,
) (client.ProofClient, error) {
	pClient := &proofClient{}

	if err := depinject.Inject(
		deps,
		&pClient.txClient,
		&pClient.txCtx,
	); err != nil {
		return nil, err
	}

	for _, opt := range opts {
		opt(pClient)
	}

	if err := pClient.validateConfigAndSetDefaults(); err != nil {
		return nil, err
	}

	return pClient, nil
}

// SubmitProofs constructs submit proof messages into a single transaction
// then signs and broadcasts it to the network via #txClient. It blocks until
// the transaction is included in a block or times out.
func (sClient *proofClient) SubmitProofs(
	ctx context.Context,
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
	//  with off-chain pkgs/nomenclature.
	eitherErr := sClient.txClient.SignAndBroadcast(ctx, msgs...)
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
				"supplier_addr": proofMsg.SupplierAddress,
				"app_addr":      sessionHeader.ApplicationAddress,
				"session_id":    sessionHeader.SessionId,
				"service":       sessionHeader.Service.Id,
			}).
			Msg("submitted a new proof")
	}

	return <-errCh
}

// CreateClaims constructs create claim messages into a single transaction
// then signs and broadcasts it to the network via #txClient. It blocks until
// the transaction is included in a block or times out.
func (sClient *proofClient) CreateClaims(
	ctx context.Context,
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
	//  with off-chain pkgs/nomenclature.
	eitherErr := sClient.txClient.SignAndBroadcast(ctx, msgs...)
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
				"supplier_addr": claimMsg.SupplierAddress,
				"app_addr":      sessionHeader.ApplicationAddress,
				"session_id":    sessionHeader.SessionId,
				"service":       sessionHeader.Service.Id,
			}).
			Msg("created a new claim")
	}

	return <-errCh
}

// Address returns an address of the supplier client.
func (sClient *proofClient) Address() *cosmostypes.AccAddress {
	return &sClient.signingKeyAddr
}

// validateConfigAndSetDefaults attempts to get the address from the keyring
// corresponding to the key whose name matches the configured signingKeyName.
// If signingKeyName is empty or the keyring does not contain the corresponding
// key, an error is returned.
func (sClient *proofClient) validateConfigAndSetDefaults() error {
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
