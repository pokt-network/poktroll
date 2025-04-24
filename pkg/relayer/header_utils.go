package relayer

import (
	"fmt"
	"github.com/pokt-network/poktroll/x/service/types"
	"net/http"
)

// ForwardIdentityHeaders adds Pocket-specific identity headers from the relay metadata
// to the HTTP request header. This helps with tracking and authentication at the
// service backend level.
func ForwardIdentityHeaders(header *http.Header, meta types.RelayRequestMetadata) {
	// Supplier identification
	header.Set("X-Pocket-Supplier", meta.SupplierOperatorAddress)

	// Service identification
	header.Set("X-Pocket-Service", meta.SessionHeader.ServiceId)

	// Application identification (if available)
	if meta.SessionHeader != nil {
		header.Set("X-Pocket-Session-Id", meta.SessionHeader.SessionId)
		header.Set("X-Pocket-Application", meta.SessionHeader.ApplicationAddress)
		header.Set("X-Pocket-Session-Start-Height", fmt.Sprintf("%d", meta.SessionHeader.SessionStartBlockHeight))
		header.Set("X-Pocket-Session-End-Height", fmt.Sprintf("%d", meta.SessionHeader.SessionEndBlockHeight))
	}
}
