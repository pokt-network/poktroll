package gateway

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1 "github.com/pokt-network/poktroll/api/poktroll/gateway"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: modulev1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Shows the parameters of the module",
				},
				{
					RpcMethod: "AllGateways",
					Use:       "list-gateway",
					Short:     "List all gateway",
				},
				{
					RpcMethod:      "Gateway",
					Use:            "show-gateway [gateway address]",
					Short:          "Shows a gateway",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "address"},
					},
				},
				// this line is used by ignite scaffolding # autocli/query
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              modulev1.Msg_ServiceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "UpdateParams",
					Skip:      true, // skipped because authority gated
				},
				{
					RpcMethod:      "StakeGateway",
					Use:            "stake-gateway --config [stake config]",
					Short:          "Stake a Gateway",
					Long: `Stake a gateway with the provided parameters. This is a broadcast operation that
will stake the tokens and associate them with the gateway specified by the 'from' address.`,
					Example: `$ poktrolld tx gateway stake-gateway --config stake_config.yaml --keyring-backend test --from $(GATEWAY) --node $(POCKET_NODE) --home=$(POKTROLLD_HOME)`,
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"from": {
							Name: "from",
							DefaultValue: "", 
							Usage: "The address of the gateway",
						},
						"config": {
							Name: "config", 
							DefaultValue: "", 
							Usage: "The path to the config file",
						},
					},
				},
				{
					RpcMethod:      "UnstakeGateway",
					Use:            "unstake-gateway",
					Short:          "Unstake a Gateway",
					Long: `Unstake a gateway. This is a broadcast operation that will unstake the gateway
specified by the 'from' address.`,
					Example:`poktrolld tx gateway unstake-gateway --keyring-backend test --from $(GATEWAY) --node $(POCKET_NODE) --home=$(POKTROLLD_HOME)`,
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"from": {
							Name: "from", 
							DefaultValue: "", 
							Usage: "The address of the gateway",
						},
					},
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
