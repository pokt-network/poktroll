package supplier

import "github.com/pokt-network/pocket/pkg/client"

// WithSigningKeyName sets the name of the operator key which the supplier
// client should retrieve from the keyring to use for authoring and signing
// CreateClaim and SubmitProof messages.
func WithSigningKeyName(keyName string) client.SupplierClientOption {
	return func(sClient client.SupplierClient) {
		sClient.(*supplierClient).signingKeyName = keyName
	}
}
