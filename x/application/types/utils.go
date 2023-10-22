package types

import (
	sdkerrors "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocdc "github.com/cosmos/cosmos-sdk/crypto/codec"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
)

// anyToPubKey converts an Any type to a cryptotypes.PubKey
func anyToPubKey(anyPk codectypes.Any) (cryptotypes.PubKey, error) {
	reg := codectypes.NewInterfaceRegistry()
	cryptocdc.RegisterInterfaces(reg)
	cdc := codec.NewProtoCodec(reg)
	var pub cryptotypes.PubKey
	if err := cdc.UnpackAny(&anyPk, &pub); err != nil {
		return nil, sdkerrors.Wrapf(ErrAppAnyIsNotPubKey, "Any is not cosmos.crypto.PubKey: got %s", anyPk.TypeUrl)
	}
	return pub, nil
}
