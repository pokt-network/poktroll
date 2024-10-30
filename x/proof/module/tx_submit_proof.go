package proof

import (
	"encoding/base64"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

// TODO_TECHDEBT(@bryanchriswhite): Add unit tests for the CLI command when implementing the business logic.

var _ = strconv.Itoa(0)

func CmdSubmitProof() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-proof <session_header> <proof_base64>",
		Short: "Broadcast message submit-proof",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			sessionHeaderEncodedStr := args[0]
			smstProofEncodedStr := args[1]

			// Get the session header
			sessionHeaderBz, err := base64.StdEncoding.DecodeString(sessionHeaderEncodedStr)
			if err != nil {
				return err
			}
			sessionHeader := &sessiontypes.SessionHeader{}
			cdc := codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
			cdc.MustUnmarshalJSON(sessionHeaderBz, sessionHeader)

			smstProof, err := base64.StdEncoding.DecodeString(smstProofEncodedStr)
			if err != nil {
				return err
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgSubmitProof(
				clientCtx.GetFromAddress().String(),
				sessionHeader,
				smstProof,
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
