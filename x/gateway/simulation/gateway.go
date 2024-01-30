package simulation

import (
	"strconv"
)

// Prevent strconv unused error
var _ = strconv.IntSize

// TODO_TECHDEBT: Adjust simulations appropriately for the exiting gateway types
// and its message handlers.
// func SimulateMsgCreateGateway(
// 	ak types.AccountKeeper,
// 	bk types.BankKeeper,
// 	k keeper.Keeper,
// ) simtypes.Operation {
// 	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
// 	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
// 		simAccount, _ := simtypes.RandomAcc(r, accs)
//
// 		i := r.Int()
// 		msg := &types.MsgCreateGateway{
// 			Address: simAccount.Address.String(),
// 			Index:   strconv.Itoa(i),
// 		}
//
// 		_, found := k.GetGateway(ctx, msg.Address)
// 		if found {
// 			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), "Gateway already exist"), nil, nil
// 		}
//
// 		txCtx := simulation.OperationInput{
// 			R:               r,
// 			App:             app,
// 			TxGen:           moduletestutil.MakeTestEncodingConfig().TxConfig,
// 			Cdc:             nil,
// 			Msg:             msg,
// 			Context:         ctx,
// 			SimAccount:      simAccount,
// 			ModuleName:      types.ModuleName,
// 			CoinsSpentInMsg: sdk.NewCoins(),
// 			AccountKeeper:   ak,
// 			Bankkeeper:      bk,
// 		}
// 		return simulation.GenAndDeliverTxWithRandFees(txCtx)
// 	}
// }
//
// func SimulateMsgUpdateGateway(
// 	ak types.AccountKeeper,
// 	bk types.BankKeeper,
// 	k keeper.Keeper,
// ) simtypes.Operation {
// 	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
// 	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
// 		var (
// 			simAccount = simtypes.Account{}
// 			gateway    = types.Gateway{}
// 			msg        = &types.MsgUpdateGateway{}
// 			allGateway = k.GetAllGateway(ctx)
// 			found      = false
// 		)
// 		for _, obj := range allGateway {
// 			simAccount, found = FindAccount(accs, obj.Address)
// 			if found {
// 				gateway = obj
// 				break
// 			}
// 		}
// 		if !found {
// 			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), "gateway address not found"), nil, nil
// 		}
// 		msg.Address = simAccount.Address.String()
//
// 		msg.Index = gateway.Index
//
// 		txCtx := simulation.OperationInput{
// 			R:               r,
// 			App:             app,
// 			TxGen:           moduletestutil.MakeTestEncodingConfig().TxConfig,
// 			Cdc:             nil,
// 			Msg:             msg,
// 			Context:         ctx,
// 			SimAccount:      simAccount,
// 			ModuleName:      types.ModuleName,
// 			CoinsSpentInMsg: sdk.NewCoins(),
// 			AccountKeeper:   ak,
// 			Bankkeeper:      bk,
// 		}
// 		return simulation.GenAndDeliverTxWithRandFees(txCtx)
// 	}
// }
//
// func SimulateMsgDeleteGateway(
// 	ak types.AccountKeeper,
// 	bk types.BankKeeper,
// 	k keeper.Keeper,
// ) simtypes.Operation {
// 	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
// 	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
// 		var (
// 			simAccount = simtypes.Account{}
// 			gateway    = types.Gateway{}
// 			msg        = &types.MsgUpdateGateway{}
// 			allGateway = k.GetAllGateway(ctx)
// 			found      = false
// 		)
// 		for _, obj := range allGateway {
// 			simAccount, found = FindAccount(accs, obj.Address)
// 			if found {
// 				gateway = obj
// 				break
// 			}
// 		}
// 		if !found {
// 			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), "gateway address not found"), nil, nil
// 		}
// 		msg.Address = simAccount.Address.String()
//
// 		msg.Index = gateway.Index
//
// 		txCtx := simulation.OperationInput{
// 			R:               r,
// 			App:             app,
// 			TxGen:           moduletestutil.MakeTestEncodingConfig().TxConfig,
// 			Cdc:             nil,
// 			Msg:             msg,
// 			Context:         ctx,
// 			SimAccount:      simAccount,
// 			ModuleName:      types.ModuleName,
// 			CoinsSpentInMsg: sdk.NewCoins(),
// 			AccountKeeper:   ak,
// 			Bankkeeper:      bk,
// 		}
// 		return simulation.GenAndDeliverTxWithRandFees(txCtx)
// 	}
// }
