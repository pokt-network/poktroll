package supplier

import (
	"context"

	"cosmossdk.io/depinject"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/smt"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/keyring"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
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

// SubmitProof constructs a submit proof message then signs and broadcasts it
// to the network via #txClient. It blocks until the transaction is included in
// a block or times out.
func (sClient *supplierClient) SubmitProof(
	ctx context.Context,
	sessionHeader sessiontypes.SessionHeader,
	proof *smt.SparseMerkleClosestProof,
) error {
	proofBz, err := proof.Marshal()
	if err != nil {
		return err
	}

	msg := &suppliertypes.MsgSubmitProof{
		SupplierAddress: sClient.signingKeyAddr.String(),
		SessionHeader:   &sessionHeader,
		Proof:           proofBz,
	}
	eitherErr := sClient.txClient.SignAndBroadcast(ctx, msg)
	err, errCh := eitherErr.SyncOrAsyncError()
	if err != nil {
		return err
	}

	return <-errCh
}

// CreateClaim constructs a creates claim message then signs and broadcasts it
// to the network via #txClient. It blocks until the transaction is included in
// a block or times out.
func (sClient *supplierClient) CreateClaim(
	ctx context.Context,
	sessionHeader sessiontypes.SessionHeader,
	rootHash []byte,
) error {
	msg := &suppliertypes.MsgCreateClaim{
		SupplierAddress: sClient.signingKeyAddr.String(),
		SessionHeader:   &sessionHeader,
		RootHash:        rootHash,
	}
	eitherErr := sClient.txClient.SignAndBroadcast(ctx, msg)
	err, errCh := eitherErr.SyncOrAsyncError()
	if err != nil {
		return err
	}

	err = <-errCh
	return err
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
