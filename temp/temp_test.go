package temp

import (
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
)

func TestTxSerializationWithUpdateParams(t *testing.T) {
	// Initialize the codec registry
	registry := types.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	// Construct the message
	msg := &authtypes.MsgUpdateParams{
		// Assuming there are fields like Creator and Params in MsgUpdateParams,
		// fill them accordingly with dummy or test values.
		Authority: "cosmos1...",
		Params:    authtypes.Params{MaxMemoCharacters: 100},
	}

	// Create an Any type for the message
	anyMsg, err := types.NewAnyWithValue(msg)
	require.NoError(t, err)

	// Construct the TxBody with the message
	txBody := &tx.TxBody{
		Messages: []*types.Any{anyMsg},
	}

	// Serialize the TxBody
	bz, err := cdc.MarshalJSON(txBody)

	fmt.Println(string(bz))
	require.NoError(t, err)

	// Deserialize to verify correctness
	var deserialized tx.TxBody
	err = cdc.Unmarshal(bz, &deserialized)
	require.NoError(t, err)

	// Check if the original message is equal to the deserialized message
	require.Equal(t, txBody, &deserialized)
}
