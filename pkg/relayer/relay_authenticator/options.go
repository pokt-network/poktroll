package relay_authenticator

import (
	"github.com/pokt-network/poktroll/pkg/relayer"
)

// WithSigningKeyNames sets the signing key names used by the relay authenticator
// to sign relay responses.
func WithSigningKeyNames(keyNames []string) relayer.RelayAuthenticatorOption {
	return func(relAuth relayer.RelayAuthenticator) {
		relAuth.(*relayAuthenticator).signingKeyNames = keyNames
	}
}
