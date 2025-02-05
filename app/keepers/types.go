package keepers

// Keepers have been moved to a separate package to ensure all keepers are accessible in `upgrades.Upgrade.CreateUpgradeHandler`.
// This allows for passing all keepers into the upgrade handler and accessing/changing blockchain state across all modules.
// When performing `ignite scaffold` the keeper will be added to `app.go`. Please move them here.
//
// For more details, refer to the comment section of this PR: https://github.com/pokt-network/poktroll/pull/702

import (
	circuitkeeper "cosmossdk.io/x/circuit/keeper"
	evidencekeeper "cosmossdk.io/x/evidence/keeper"
	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"
	upgradekeeper "cosmossdk.io/x/upgrade/keeper"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	consensuskeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	crisiskeeper "github.com/cosmos/cosmos-sdk/x/crisis/keeper"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	groupkeeper "github.com/cosmos/cosmos-sdk/x/group/keeper"
	mintkeeper "github.com/cosmos/cosmos-sdk/x/mint/keeper"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	icacontrollerkeeper "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/keeper"
	icahostkeeper "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/keeper"
	ibcfeekeeper "github.com/cosmos/ibc-go/v8/modules/apps/29-fee/keeper"
	ibctransferkeeper "github.com/cosmos/ibc-go/v8/modules/apps/transfer/keeper"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"

	applicationmodulekeeper "github.com/pokt-network/poktroll/x/application/keeper"
	gatewaymodulekeeper "github.com/pokt-network/poktroll/x/gateway/keeper"
	migrationmodulekeeper "github.com/pokt-network/poktroll/x/migration/keeper"
	proofmodulekeeper "github.com/pokt-network/poktroll/x/proof/keeper"
	servicemodulekeeper "github.com/pokt-network/poktroll/x/service/keeper"
	sessionmodulekeeper "github.com/pokt-network/poktroll/x/session/keeper"
	sharedmodulekeeper "github.com/pokt-network/poktroll/x/shared/keeper"
	suppliermodulekeeper "github.com/pokt-network/poktroll/x/supplier/keeper"
	tokenomicsmodulekeeper "github.com/pokt-network/poktroll/x/tokenomics/keeper"
)

// Keepers includes all possible keepers. We separated it into a separate struct to make it easier to scaffold upgrades.
type Keepers struct {
	AccountKeeper         authkeeper.AccountKeeper
	BankKeeper            bankkeeper.Keeper
	StakingKeeper         *stakingkeeper.Keeper
	SlashingKeeper        slashingkeeper.Keeper
	MintKeeper            mintkeeper.Keeper
	DistrKeeper           distrkeeper.Keeper
	GovKeeper             *govkeeper.Keeper
	CrisisKeeper          *crisiskeeper.Keeper
	UpgradeKeeper         *upgradekeeper.Keeper
	ParamsKeeper          paramskeeper.Keeper
	AuthzKeeper           authzkeeper.Keeper
	EvidenceKeeper        evidencekeeper.Keeper
	FeeGrantKeeper        feegrantkeeper.Keeper
	GroupKeeper           groupkeeper.Keeper
	ConsensusParamsKeeper consensuskeeper.Keeper
	CircuitBreakerKeeper  circuitkeeper.Keeper

	// IBC
	IBCKeeper           *ibckeeper.Keeper // IBC Keeper must be a pointer in the app, so we can SetRouter on it correctly
	CapabilityKeeper    *capabilitykeeper.Keeper
	IBCFeeKeeper        ibcfeekeeper.Keeper
	ICAControllerKeeper icacontrollerkeeper.Keeper
	ICAHostKeeper       icahostkeeper.Keeper
	TransferKeeper      ibctransferkeeper.Keeper

	// Scoped IBC
	ScopedIBCKeeper           capabilitykeeper.ScopedKeeper
	ScopedIBCTransferKeeper   capabilitykeeper.ScopedKeeper
	ScopedICAControllerKeeper capabilitykeeper.ScopedKeeper
	ScopedICAHostKeeper       capabilitykeeper.ScopedKeeper

	// Pocket specific keepers
	ServiceKeeper     servicemodulekeeper.Keeper
	GatewayKeeper     gatewaymodulekeeper.Keeper
	ApplicationKeeper applicationmodulekeeper.Keeper
	SupplierKeeper    suppliermodulekeeper.Keeper
	SessionKeeper     sessionmodulekeeper.Keeper
	ProofKeeper       proofmodulekeeper.Keeper
	TokenomicsKeeper  tokenomicsmodulekeeper.Keeper
	SharedKeeper      sharedmodulekeeper.Keeper
	MigrationKeeper   migrationmodulekeeper.Keeper
}
