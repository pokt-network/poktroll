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
			// REQUIRED: the module provides its own GetQueryCmd (for `card`), and without
			// this flag autocli uses that custom command INSTEAD of the generated ones --
			// silently deleting show-service, all-services, compute-units-per-relay-* and
			// relay-mining-difficulty-* from the CLI.
			// Ref: cosmossdk.io/client/v2/autocli enhanceCustomCmd.
			EnhanceCustomCommand: true,
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

By default, service metadata (API specifications) is excluded to reduce payload size.
Use --dehydrated=false to include full metadata for all services.

Supports pagination via flags if there are many services.`,
					Example: `pocketd q service all-services
pocketd q service all-services --limit 50
pocketd q service all-services --page 2
pocketd q service all-services --dehydrated=false`,
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"dehydrated": {
							Name:         "dehydrated",
							Usage:        "Exclude service metadata to reduce payload size (default true)",
							DefaultValue: "true",
						},
					},
				},
				{
					RpcMethod: "Service",
					Use:       "show-service [service-id]",
					Short:     "Show full details for a specific service",
					Long: `Retrieves complete service information by its unique on-chain ID.

Returns all service details including:
- Service ID, name, and compute units per relay
- Owner address
- Full metadata (API specifications up to 256 KiB) by default

Use the --dehydrated flag to exclude metadata and reduce payload size.`,
					Example: `pocketd q service show-service pocket
pocketd q service show-service anvil --output json
pocketd q service show-service pocket --dehydrated`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"dehydrated": {
							Name:  "dehydrated",
							Usage: "Exclude service metadata to reduce payload size",
						},
					},
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
				{
					RpcMethod: "RelayMiningDifficultyAtHeight",
					Use:       "relay-mining-difficulty-at-height [service-id] [block-height]",
					Short:     "Show relay mining difficulty for a service at a specific block height",
					Long: `
- Shows the relay mining difficulty that was effective for a specific service at a given block height.
- Useful for historical analysis and verifying difficulty used during past sessions.
`,
					Example:        `pocketd q service relay-mining-difficulty-at-height <service-id> 1000`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "serviceId"}, {ProtoField: "blockHeight"}},
				},
				{
					RpcMethod: "RelayMiningDifficultyHistory",
					Use:       "relay-mining-difficulty-history [service-id]",
					Short:     "List the history of relay mining difficulty changes for a service",
					Long: `
- Lists all historical relay mining difficulty changes for a specific service.
- Each entry shows when a difficulty became effective and the difficulty parameters.
- Supports pagination via flags if there are many entries.
`,
					Example:        `pocketd q service relay-mining-difficulty-history <service-id>`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "serviceId"}},
				},
				{
					RpcMethod: "ComputeUnitsPerRelayAtHeight",
					Use:       "compute-units-per-relay-at-height [service-id] [block-height]",
					Short:     "Show the compute units per relay (cupr) effective for a service at a specific block height",
					Long: `
- Shows the compute_units_per_relay that was effective for a specific service at a given block height.
- This is the value claim validation pins to the session-start height, so it is useful for verifying the cupr used during past sessions.
`,
					Example:        `pocketd q service compute-units-per-relay-at-height <service-id> 1000`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "serviceId"}, {ProtoField: "blockHeight"}},
				},
				{
					RpcMethod: "ComputeUnitsPerRelayHistory",
					Use:       "compute-units-per-relay-history [service-id]",
					Short:     "List the history of compute units per relay (cupr) changes for a service",
					Long: `
- Lists all historical compute_units_per_relay changes for a specific service.
- Each entry shows when a cupr became effective and its value.
- Supports pagination via flags if there are many entries.
`,
					Example:        `pocketd q service compute-units-per-relay-history <service-id>`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "serviceId"}},
				},
				{
					RpcMethod: "SupplierAllowlist",
					Use:       "supplier-allowlist [service-id]",
					Short:     "List the supplier allowlist of a service",
					Long: `
- Lists the addresses (supplier owner or operator) allowed to serve a service.
- A service is permissioned while its allowlist is non-empty; an empty list means any supplier may serve it.
- Supports pagination via flags if there are many entries.
`,
					Example:        `pocketd q service supplier-allowlist <service-id>`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "service_id"}},
				},
				{
					RpcMethod: "SupplierAllowlistEntry",
					Use:       "is-supplier-allowed [service-id] [address]",
					Short:     "Show whether an address is on a service's supplier allowlist",
					Long: `
- Shows whether an address (supplier owner or operator) is on a service's supplier allowlist.
- Also shows whether the service is permissioned, that is, whether its allowlist is non-empty.
- The address must be in canonical (lowercase) bech32 form.
`,
					Example:        `pocketd q service is-supplier-allowed <service-id> <address>`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "service_id"}, {ProtoField: "address"}},
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
				{
					RpcMethod: "TransferService",
					Use:       "transfer-service <service-id> <new-owner-address>",
					Short:     "Transfer ownership of a service to a new address",
					Long: `Transfer ownership of an on-chain service to a new address.
Only the current owner (--from) can initiate the transfer.
The new owner immediately gains full control of the service.`,
					Example:        `pocketd tx service transfer-service svc-foo pokt1newowner... --from currentowner`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "service_id"}, {ProtoField: "new_owner_address"}},
				},
				{
					RpcMethod: "UpdateServiceSupplierAllowlist",
					Use:       "update-supplier-allowlist <service-id>",
					Short:     "Add and/or remove suppliers on a service's supplier allowlist",
					Long: `Add and/or remove addresses (supplier owner or operator) on a service's supplier allowlist.
Only the service owner (--from) can update the allowlist.

A service is permissioned while its allowlist is non-empty: only listed suppliers may serve it.
New stakes by unlisted suppliers are rejected at once. Already-staked unlisted suppliers are deactivated for
the service at the second session start after the first entry on an open service (finish the list within the current session),
and at the next session start after a later removal. Current sessions are never affected.

Addresses must be canonical (lowercase) bech32. Pass comma-separated lists to --add and --remove (at most 100 entries combined).`,
					Example:        `pocketd tx service update-supplier-allowlist svc-foo --add pokt1abc...,pokt1def... --remove pokt1ghi... --from owner`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "service_id"}},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"add":    {Usage: "Comma-separated canonical addresses to add to the allowlist"},
						"remove": {Usage: "Comma-separated canonical addresses to remove from the allowlist"},
					},
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
