package miner

import (
	"context"

	"cosmossdk.io/depinject"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/pkg/either"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/observable/filter"
	"github.com/pokt-network/poktroll/pkg/observable/logging"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/proto/types/service"
)

var _ relayer.Miner = (*miner)(nil)

// Miner is responsible for observing servedRelayObs, hashing and checking the
// difficulty of each, finally publishing those with sufficient difficulty to
// minedRelayObs as they are applicable for relay volume.
type miner struct {
	// proofQueryClient is used to query for the minimum relay difficulty.
	proofQueryClient client.ProofQueryClient

	// relayDifficultyBits is the minimum difficulty that a relay must have to be
	// volume / reward applicable.
	relayDifficultyBits uint64
}

// NewMiner creates a new miner from the given dependencies and options. It
// returns an error if it has not been sufficiently configured or supplied.
//
// Required Dependencies:
// - ProofQueryClient
//
// Available options:
//   - WithDifficulty
func NewMiner(
	deps depinject.Config,
	opts ...relayer.MinerOption,
) (*miner, error) {
	mnr := &miner{}

	if err := depinject.Inject(deps, &mnr.proofQueryClient); err != nil {
		return nil, err
	}

	for _, opt := range opts {
		opt(mnr)
	}

	if err := mnr.setDefaults(); err != nil {
		return nil, err
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
	relaysObs := observable.Observable[*service.Relay](servedRelaysObs)

	// Map servedRelaysObs to a new observable of an either type, populated with
	// the minedRelay or an error. It is notified after the relay has been mined
	// or an error has been encountered, respectively.
	eitherMinedRelaysObs := channel.Map(ctx, relaysObs, mnr.mapMineRelay)
	logging.LogErrors(ctx, filter.EitherError(ctx, eitherMinedRelaysObs))

	return filter.EitherSuccess(ctx, eitherMinedRelaysObs)
}

// setDefaults ensures that the miner has been configured with a hasherConstructor and uses
// the default hasherConstructor if not.
func (mnr *miner) setDefaults() error {
	ctx := context.TODO()
	params, err := mnr.proofQueryClient.GetParams(ctx)
	if err != nil {
		return err
	}

	if mnr.relayDifficultyBits == 0 {
		mnr.relayDifficultyBits = params.GetMinRelayDifficultyBits()
	}
	return nil
}

// mapMineRelay is intended to be used as a MapFn.
// 1. It hashes the relay and compares its difficult to the minimum threshold.
// 2. If the relay difficulty is sufficient -> return an Either[MineRelay Value]
// 3. If an error is encountered -> return an Either[error]
// 4. Otherwise, skip the relay.
func (mnr *miner) mapMineRelay(
	_ context.Context,
	relay *service.Relay,
) (_ either.Either[*relayer.MinedRelay], skip bool) {
	// TODO_TECHDEBT(@red-0ne, #446): Centralize the configuration for the SMT spec.
	// TODO_TECHDEBT(@red-0ne): marshal using canonical codec.
	relayBz, err := relay.Marshal()
	if err != nil {
		return either.Error[*relayer.MinedRelay](err), false
	}
	relayHash := service.GetHashFromBytes(relayBz)

	// The relay IS NOT volume / reward applicable
	if uint64(protocol.CountHashDifficultyBits(relayHash)) < mnr.relayDifficultyBits {
		return either.Success[*relayer.MinedRelay](nil), true
	}

	// The relay IS volume / reward applicable
	return either.Success(&relayer.MinedRelay{
		Relay: *relay,
		Bytes: relayBz,
		Hash:  relayHash[:],
	}), false
}
