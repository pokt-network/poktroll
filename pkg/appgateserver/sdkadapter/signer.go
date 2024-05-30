package sdkadapter

import (
	"encoding/hex"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/pokt-network/shannon-sdk/sdk"
)

var _ sdk.Signer = (*signer)(nil)

// signer is a struct that caches and returns the signing key in hex format.
type signer struct {
	signingKey string
}

// NewSigner creates a new ShannonSDK compatible signer with the given signing key.
func NewSigner(signingKey cryptotypes.PrivKey) (sdk.Signer, error) {
	signingKeyHex := hex.EncodeToString(signingKey.Bytes())
	signer := &signer{
		signingKey: signingKeyHex,
	}

	return signer, nil
}

// GetPrivateKeyHex returns the AppGateServer signing key in hex format.
func (s *signer) GetPrivateKeyHex() string {
	return s.signingKey
}
