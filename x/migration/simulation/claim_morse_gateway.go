package simulation

import (
	"math/rand"

	"github.com/pokt-network/poktroll/x/migration/keeper"
	"github.com/pokt-network/poktroll/x/migration/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
)

func SimulateMsgClaimMorseGateway(
	ak types.AccountKeeper,
	bk types.BankKeeper,
	k keeper.Keeper,
) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		msg := &types.MsgClaimMorseGateway{
			ShannonDestAddress: simAccount.Address.String(),
		}

		// TODO: Handling the ClaimMorseGateway simulation

		return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), "ClaimMorseGateway simulation not implemented"), nil, nil
	}
}
