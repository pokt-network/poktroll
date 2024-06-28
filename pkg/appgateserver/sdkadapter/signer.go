package sdkadapter

import (
	"encoding/hex"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/pokt-network/shannon-sdk"
)

// NewSigner creates a new ShannonSDK compatible signer with the given signing key.
func NewSigner(signingKey cryptotypes.PrivKey) (*sdk.Signer, error) {
	PrivateKeyHex := hex.EncodeToString(signingKey.Bytes())
	signer := &sdk.Signer{
		PrivateKeyHex: PrivateKeyHex,
	}

	return signer, nil
}
