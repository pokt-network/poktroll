package supplier

import (
	"pocket/pkg/client"
)

func WithSigningKeyName(keyName string) client.SupplierClientOption {
	return func(sClient client.SupplierClient) {
		sClient.(*supplierClient).signingKeyName = keyName
	}
}
