package integration_test

import (
	"testing"

	"github.com/pokt-network/poktroll/cmd/poktrolld/cmd"
)

func init() {
	cmd.InitSDKConfig()
}

func TestUpdateRelayMiningDifficulty_NewServiceSeenForTheFirstTime(t *testing.T) {}

func UpdateRelayMiningDifficulty_UpdatingMultipleServicesAtOnce(t *testing.T) {}

func UpdateRelayMiningDifficulty_UpdateServiceIsNotSeenForAWhile(t *testing.T) {}

func UpdateRelayMiningDifficulty_UpdateServiceIsIncreasing(t *testing.T) {}

func UpdateRelayMiningDifficulty_UpdateServiceIsDecreasing(t *testing.T) {}