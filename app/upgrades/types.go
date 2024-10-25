package upgrades

import (
	"time"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/authz"

	"github.com/pokt-network/poktroll/app/keepers"
)

var testNetPnfAddress = "pokt1r6ja6rz6rpae58njfrsgs5n5sp3r36r2q9j04h"
var testNetAuthorityAddress = "pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t"

// Upgrade represents a protocol upgrade in code.
// Once a `MsgSoftwareUpgrade` is submitted on-chain, and `Upgrade.PlanName` matches the `Plan.Name`,
// the upgrade will be scheduled for execution at the corresponding height.
type Upgrade struct {
	// PlanName is a name an upgrade is matched to from the on-chain `upgradetypes.Plan`.
	PlanName string

	// CreateUpgradeHandler returns an upgrade handler that will be executed at the time of the upgrade.
	// State changes and protocol version upgrades should be performed here.
	CreateUpgradeHandler func(*module.Manager, *keepers.Keepers, module.Configurator) upgradetypes.UpgradeHandler

	// StoreUpgrades adds, renames and deletes KVStores in the state to prepare for a protocol upgrade.
	StoreUpgrades storetypes.StoreUpgrades
}

type grantAuthorization struct {
	grantee       sdk.AccAddress
	granter       sdk.AccAddress
	authorization authz.Authorization
	expiration    *time.Time
}

func newTestNetGrantAuthorization(msg string) grantAuthorization {
	authorization := authz.NewGenericAuthorization(msg)
	expiration, err := time.Parse(time.RFC3339, "2500-01-01T00:00:00Z")
	if err != nil {
		panic(err)
	}
	err = authorization.ValidateBasic()
	if err != nil {
		panic(err)
	}
	return grantAuthorization{
		grantee:       sdk.MustAccAddressFromBech32(testNetAuthorityAddress),
		granter:       sdk.MustAccAddressFromBech32(testNetPnfAddress),
		authorization: authorization,
		expiration:    &expiration,
	}
}
