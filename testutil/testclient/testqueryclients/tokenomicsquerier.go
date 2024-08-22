package testqueryclients

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/pokt-network/poktroll/testutil/mockclient"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// relayDifficultyTargets is a map of: serviceId -> RelayMiningDifficulty
// It is updated by the SetServiceRelayDifficultyTargetHash, and read by
// the mock tokenomics query client to get a specific service's relay difficulty
// target hash.
var relayDifficultyTargets map[string]*tokenomicstypes.RelayMiningDifficulty

func init() {
	relayDifficultyTargets = make(map[string]*tokenomicstypes.RelayMiningDifficulty)
}

// NewTestTokenomicsQueryClient creates a mock of the TokenomicsQueryClient
// which allows the caller to call GetSession any times and will return
// the session matching the app address, serviceID and the blockHeight passed.
func NewTestTokenomicsQueryClient(
	t *testing.T,
) *mockclient.MockTokenomicsQueryClient {
	t.Helper()
	ctrl := gomock.NewController(t)

	tokenomicsQuerier := mockclient.NewMockTokenomicsQueryClient(ctrl)
	tokenomicsQuerier.EXPECT().GetServiceRelayDifficultyTargetHash(gomock.Any(), gomock.Any()).
		DoAndReturn(func(
			_ context.Context,
			serviceId string,
		) (*tokenomicstypes.RelayMiningDifficulty, error) {
			relayDifficulty, ok := relayDifficultyTargets[serviceId]
			if !ok {
				return nil, tokenomicstypes.ErrTokenomicsMissingRelayMiningDifficulty.Wrapf("retrieving the relay mining difficulty for service %s", serviceId)
			}

			return relayDifficulty, nil
		}).
		AnyTimes()

	return tokenomicsQuerier
}

// AddServiceRelayDifficultyTargetHash sets the relay difficulty target hash
// for the given service to mock it "existing" on chain.
// It will also remove the service relay difficulty target hashes from the map when the test is cleaned up.
func SetServiceRelayDifficultyTargetHash(t *testing.T,
	serviceId string,
	relayDifficultyTargetHash []byte,
) {
	t.Helper()

	relayDifficultyTargets[serviceId] = &tokenomicstypes.RelayMiningDifficulty{
		ServiceId:  serviceId,
		TargetHash: relayDifficultyTargetHash,
	}

	t.Cleanup(func() {
		delete(relayDifficultyTargets, serviceId)
	})
}
