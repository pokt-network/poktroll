package supplier

import (
	"github.com/pokt-network/poktroll/pkg/client"
)

func WithSigningKeyName(keyName string) client.SupplierClientOption {
	return func(sClient client.SupplierClient) {
		sClient.(*supplierClient).signingKeyName = keyName
	}
}
