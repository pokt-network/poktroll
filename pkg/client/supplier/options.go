package supplier

import "github.com/pokt-network/poktroll/pkg/client"

// WithSigningKeyName sets the name of the key which the supplier client should
// retrieve from the keyring to use for authoring and signing CreateClaim and
// SubmitProof messages.
func WithSigningKeyNames(keyNames []string) client.SupplierClientOption {
	return func(sClient client.SupplierClient) {
		sClient.(*supplierClient).signingKeyNames = keyNames
	}
}
