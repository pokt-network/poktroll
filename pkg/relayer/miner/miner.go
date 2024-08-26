package miner

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/depinject"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/pkg/either"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/observable/filter"
	"github.com/pokt-network/poktroll/pkg/observable/logging"
	"github.com/pokt-network/poktroll/pkg/relayer"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
)

var _ relayer.Miner = (*miner)(nil)

// Miner is responsible for observing servedRelayObs, hashing and checking the
// difficulty of each, finally publishing those with sufficient difficulty to
// minedRelayObs as they are applicable for relay volume.
type miner struct {
	// tokenomicsQueryClient is used to query for the relay difficulty target hash of a service.
	// relay_difficulty is the target hash which a relay hash must be less than to be volume/reward applicable.
	tokenomicsQueryClient client.TokenomicsQueryClient
}

// NewMiner creates a new miner from the given dependencies and options. It
// returns an error if it has not been sufficiently configured or supplied.
//
// Required Dependencies:
// - ProofQueryClient
//
// Available options:
//   - WithRelayDifficultyTargetHash
func NewMiner(
	deps depinject.Config,
	opts ...relayer.MinerOption,
) (*miner, error) {
	mnr := &miner{}

	if err := depinject.Inject(deps, &mnr.tokenomicsQueryClient); err != nil {
		return nil, err
	}

	for _, opt := range opts {
		opt(mnr)
	}

	return mnr, nil
}

// MinedRelays maps servedRelaysObs through a pipeline which:
// 1. Hashes the relay
// 2. Checks if it's above the mining difficulty
// 3. Adds it to the session tree if so
// It DOES NOT BLOCK as map operations run in their own goroutines.
func (mnr *miner) MinedRelays(
	ctx context.Context,
	servedRelaysObs relayer.RelaysObservable,
) relayer.MinedRelaysObservable {
	// NB: must cast back to generic observable type to use with Map.
	// relayer.RelaysObervable cannot be an alias due to gomock's lack of
	// support for generic types.
	relaysObs := observable.Observable[*servicetypes.Relay](servedRelaysObs)

	// Map servedRelaysObs to a new observable of an either type, populated with
	// the minedRelay or an error. It is notified after the relay has been mined
	// or an error has been encountered, respectively.
	eitherMinedRelaysObs := channel.Map(ctx, relaysObs, mnr.mapMineRelay)
	logging.LogErrors(ctx, filter.EitherError(ctx, eitherMinedRelaysObs))

	return filter.EitherSuccess(ctx, eitherMinedRelaysObs)
}

// mapMineRelay is intended to be used as a MapFn.
// 1. It hashes the relay and compares its difficulty to the minimum threshold.
// 2. If the relay difficulty is sufficient -> return an Either[MineRelay Value]
// 3. If an error is encountered -> return an Either[error]
// 4. Otherwise, skip the relay.
func (mnr *miner) mapMineRelay(
	ctx context.Context,
	relay *servicetypes.Relay,
) (_ either.Either[*relayer.MinedRelay], skip bool) {
	relayBz, err := relay.Marshal()
	if err != nil {
		return either.Error[*relayer.MinedRelay](err), false
	}
	relayHashArr := protocol.GetRelayHashFromBytes(relayBz)
	relayHash := relayHashArr[:]

	relayDifficultyTargetHash, err := mnr.getServiceRelayDifficultyTargetHash(ctx, relay.Req)
	if err != nil {
		return either.Error[*relayer.MinedRelay](err), false
	}

	// The relay IS NOT volume / reward applicable
	if !protocol.IsRelayVolumeApplicable(relayHash, relayDifficultyTargetHash) {
		return either.Success[*relayer.MinedRelay](nil), true
	}

	// The relay IS volume / reward applicable
	return either.Success(&relayer.MinedRelay{
		Relay: *relay,
		Bytes: relayBz,
		Hash:  relayHash,
	}), false
}

// getServiceRelayDifficultyTargetHash returns the relay difficulty target hash for the service referenced by the relay.
// If the service does not have a relay difficulty target hash defined, the default difficulty target hash is returned.
func (mnr *miner) getServiceRelayDifficultyTargetHash(ctx context.Context, req *servicetypes.RelayRequest) ([]byte, error) {
	if req == nil {
		return nil, errors.New("relay request is nil")
	}

	meta := req.GetMeta()
	sessionHeader := meta.GetSessionHeader()
	if sessionHeader == nil {
		return nil, errors.New("relay metadata has nil session header")
	}

	if err := sessionHeader.ValidateBasic(); err != nil {
		return nil, fmt.Errorf("invalid session header: %w", err)
	}

	serviceRelayDifficulty, err := mnr.tokenomicsQueryClient.GetServiceRelayDifficultyTargetHash(ctx, sessionHeader.Service.Id)
	if err != nil {
		return nil, err
	}

	return serviceRelayDifficulty.GetTargetHash(), nil
}
