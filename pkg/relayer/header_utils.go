package relayer

import (
	"fmt"
	"net/http"

	"github.com/pokt-network/poktroll/x/service/types"
)

// ForwardPocketHeaders adds Pocket-specific identity headers from the relay metadata
// to the HTTP request header. This helps with tracking and authentication at the
// service backend level.
func ForwardPocketHeaders(header *http.Header, meta types.RelayRequestMetadata) {
	// Supplier identification
	header.Set("Pocket-Supplier", meta.SupplierOperatorAddress)

	// Application and service identification (if available)
	if meta.SessionHeader != nil {
		header.Set("Pocket-Service", meta.SessionHeader.ServiceId)
		header.Set("Pocket-Session-Id", meta.SessionHeader.SessionId)
		header.Set("Pocket-Application", meta.SessionHeader.ApplicationAddress)
		header.Set("Pocket-Session-Start-Height", fmt.Sprintf("%d", meta.SessionHeader.SessionStartBlockHeight))
		header.Set("Pocket-Session-End-Height", fmt.Sprintf("%d", meta.SessionHeader.SessionEndBlockHeight))
	}
}
