package service

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	servicetypes "github.com/pokt-network/poktroll/x/service/types"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: servicetypes.Query_serviceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Shows the parameters of the service module",
					Long: `
- Display all on-chain parameters for the service module.
- Useful for debugging and configuration introspection.
`,
					Example: `pocketd q service params`,
				},
				{
					RpcMethod: "AllServices",
					Use:       "all-services",
					Short:     "List all services registered on-chain",
					Long: `Lists all services currently registered in the network.

NOTE: Service metadata (API specifications) is excluded from list queries
to reduce payload size. Use 'show-service' to retrieve full service details
including metadata for a specific service.

Supports pagination via flags if there are many services.`,
					Example: `pocketd q service all-services
pocketd q service all-services --limit 50
pocketd q service all-services --page 2`,
				},
				{
					RpcMethod: "Service",
					Use:       "show-service [service-id]",
					Short:     "Show full details for a specific service",
					Long: `Retrieves complete service information by its unique on-chain ID.

Returns all service details including:
- Service ID, name, and compute units per relay
- Owner address
- Full metadata (API specifications up to 256 KiB)

This is the only query that returns service metadata. Use 'all-services'
for a lightweight list without metadata.`,
					Example:        `pocketd q service show-service pocket
pocketd q service show-service anvil --output json`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
				},
				{
					RpcMethod: "RelayMiningDifficultyAll",
					Use:       "relay-mining-difficulty-all",
					Short:     "List relay mining difficulty for all services",
					Long: `
- Lists the relay mining difficulty for every service.
- Useful for monitoring and network analytics.
`,
					Example: `pocketd q service relay-mining-difficulty-all`,
				},
				{
					RpcMethod: "RelayMiningDifficulty",
					Use:       "relay-mining-difficulty [service-id]",
					Short:     "Show relay mining difficulty for a service",
					Long: `
- Shows the relay mining difficulty for a specific service.
- Use this to check the current mining target for a given service.
`,
					Example:        `pocketd q service relay-mining-difficulty <service-id>`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "serviceId"}},
				},
				// this line is used by ignite scaffolding # autocli/query
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              servicetypes.Msg_serviceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "UpdateParams",
					Skip:      true, // skipped because authority gated
				},
				{
					RpcMethod: "UpdateParam",
					Skip:      true, // skipped because authority gated
				},
				{
					RpcMethod: "AddService",
					Use:       "add-service <service-id> <service-description> <compute-units-per-relay>",
					Short:     "Create a new service on-chain.",
					Long: `
- Register a new service specifying:
  - <service-id>: unique string (max 42 chars)
  - <service-description>: description (max 169 chars)
  - <compute-units-per-relay>: integer value`,
					Example:        `pocketd tx service add-service svc-foo "service description" 13 --fees 300upokt --from foo`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						// {ProtoField: "serviceId"},
						// {ProtoField: "description"},
						// {ProtoField: "computeUnitsPerRelay"},
					},
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
