package simulation

import (
	"fmt"
	"math/rand"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"

	"github.com/pokt-network/poktroll/x/service/keeper"
	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func SimulateMsgAddService(
	ak types.AccountKeeper,
	bk types.BankKeeper,
	k keeper.Keeper,
) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		rndNum := r.Intn(100)
		msg := &types.MsgAddService{
			Address: simAccount.Address.String(),
			Service: sharedtypes.Service{
				Id:   fmt.Sprintf("svcId%d", rndNum),
				Name: fmt.Sprintf("svcName%d", rndNum),
			},
		}

		// TODO: Handling the AddService simulation

		return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "AddService simulation not implemented"), nil, nil
	}
}
