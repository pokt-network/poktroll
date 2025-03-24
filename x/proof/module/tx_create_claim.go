package proof

import (
	"encoding/base64"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/spf13/cobra"

	"github.com/pokt-network/pocket/x/proof/types"
	sessiontypes "github.com/pokt-network/pocket/x/session/types"
)

// TODO_TEST(@bryanchriswhite): Add unit tests for the CLI command when implementing the business logic.

func CmdCreateClaim() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-claim <session_header> <root_hash_base64>",
		Short: "Broadcast message create-claim",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			sessionHeaderEncodedStr := args[0]
			rootHashEncodedStr := args[1]

			// Get the session header
			cdc := codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
			sessionHeaderBz, err := base64.StdEncoding.DecodeString(sessionHeaderEncodedStr)
			if err != nil {
				return err
			}
			sessionHeader := sessiontypes.SessionHeader{}
			cdc.MustUnmarshalJSON(sessionHeaderBz, &sessionHeader)

			// Get the root hash
			rootHash, err := base64.StdEncoding.DecodeString(rootHashEncodedStr)
			if err != nil {
				return err
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgCreateClaim(
				clientCtx.GetFromAddress().String(),
				&sessionHeader,
				rootHash,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
