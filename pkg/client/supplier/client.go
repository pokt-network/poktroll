package supplier

import (
	"context"

	"cosmossdk.io/depinject"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/keyring"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/relayer"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
)

var _ client.SupplierClient = (*supplierClient)(nil)

// supplierClient
type supplierClient struct {
	signingKeyName string
	signingKeyAddr cosmostypes.AccAddress

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
	sessionProofs []*relayer.SessionProof,
) error {
	logger := polylog.Ctx(ctx)

	msgs := make([]cosmostypes.Msg, len(sessionProofs))

	for i, sessionProof := range sessionProofs {
		// TODO(@bryanchriswhite): reconcile splitting of supplier & proof modules
		//  with off-chain pkgs/nomenclature.
		msgs[i] = &prooftypes.MsgSubmitProof{
			SupplierAddress: sessionProof.SupplierAddress.String(),
			SessionHeader:   sessionProof.SessionHeader,
			Proof:           sessionProof.ProofBz,
		}
	}

	eitherErr := sClient.txClient.SignAndBroadcast(ctx, msgs...)
	err, errCh := eitherErr.SyncOrAsyncError()
	if err != nil {
		return err
	}

	for _, sessionProof := range sessionProofs {
		sessionHeader := sessionProof.SessionHeader
		// TODO_IMPROVE: log details related to what & how much is being proven
		logger.Info().
			Fields(map[string]any{
				"supplier_addr": sessionProof.SupplierAddress.String(),
				"app_addr":      sessionHeader.ApplicationAddress,
				"session_id":    sessionHeader.SessionId,
				"service":       sessionHeader.Service.Id,
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
	sessionClaims []*relayer.SessionClaim,
) error {
	logger := polylog.Ctx(ctx)

	msgs := make([]cosmostypes.Msg, len(sessionClaims))

	// TODO(@bryanchriswhite): reconcile splitting of supplier & proof modules
	//  with off-chain pkgs/nomenclature.
	for i, sessionClaim := range sessionClaims {
		msgs[i] = &prooftypes.MsgCreateClaim{
			SupplierAddress: sessionClaim.SupplierAddress.String(),
			SessionHeader:   sessionClaim.SessionHeader,
			RootHash:        sessionClaim.RootHash,
		}
	}
	eitherErr := sClient.txClient.SignAndBroadcast(ctx, msgs...)
	err, errCh := eitherErr.SyncOrAsyncError()
	if err != nil {
		return err
	}

	for _, claim := range sessionClaims {
		sessionHeader := claim.SessionHeader
		// TODO_IMPROVE: log details related to how much is claimed
		logger.Info().
			Fields(map[string]any{
				"supplier_addr": claim.SupplierAddress.String(),
				"app_addr":      sessionHeader.ApplicationAddress,
				"session_id":    sessionHeader.SessionId,
				"service":       sessionHeader.Service.Id,
			}).
			Msg("created a new claim")
	}

	return <-errCh
}

// Address returns an address of the supplier client.
func (sClient *supplierClient) Address() *cosmostypes.AccAddress {
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
