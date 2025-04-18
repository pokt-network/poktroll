package upgrades

import (
	"context"
	"fmt"
	"time"

	"cosmossdk.io/log"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"

	"github.com/pokt-network/poktroll/app/keepers"
)

func applyNewAuthorizations(
	ctx context.Context,
	keepers *keepers.Keepers,
	upgradeLogger log.Logger,
	grantAuthorizationMessages []string,
) (err error) {
	logger := upgradeLogger.With("method", "applyNewAuthorizations")
	logger.Info("Starting authorization updates")

	expiration, err := time.Parse(time.RFC3339, "2500-01-01T00:00:00Z")
	if err != nil {
		return fmt.Errorf("failed to parse time: %w", err)
	}

	// Get the granter address of the migration module (i.e. authority)
	granterAddr := keepers.MigrationKeeper.GetAuthority()
	granterCosmosAddr, err := keepers.AccountKeeper.AddressCodec().StringToBytes(granterAddr)
	if err != nil {
		panic(err)
	}

	// Get the grantee address for the current network (i.e. pnf or grove)
	granteeAddr := NetworkAuthzGranteeAddress[cosmosTypes.UnwrapSDKContext(ctx).ChainID()]
	granteeCosmosAddr, err := keepers.AccountKeeper.AddressCodec().StringToBytes(granteeAddr)
	if err != nil {
		panic(err)
	}

	// Save a separate grant for each new authorization
	for _, msg := range grantAuthorizationMessages {
		err = keepers.AuthzKeeper.SaveGrant(
			ctx,
			granteeCosmosAddr,
			granterCosmosAddr,
			authz.NewGenericAuthorization(msg),
			&expiration,
		)
		if err != nil {
			return fmt.Errorf("failed to save grant for message %s: %w", msg, err)
		}
		logger.Info(fmt.Sprintf("Generic authorization granted for message %s from %s to %s", msg, granterAddr, granteeAddr))
	}

	logger.Info("Successfully finished authorization updates")

	return
}
