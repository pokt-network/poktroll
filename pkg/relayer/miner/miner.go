// Package miner encapsulates the responsibilities of the relayer miner interface:
//  1. Mining relays: Served relays are hashed and difficulty is checked.
//     Those with sufficient difficulty are added to the session SMST (tree)
//     to be applicable for relay volume.
//  2. Creating claims: The session SMST is flushed and an on-chain
//     claim is created to the amount of work done by committing
//     the tree's root.
//  3. Submitting proofs: A pseudo-random branch from the session SMST
//     is "requested" (through on-chain mechanisms) and the necessary proof
//     is submitted on-chain.
//
// This is largely accomplished by pipelining observables of relays and sessions
// Through a series of map operations.
//
// TODO_TECHDEBT: add architecture diagrams covering observable flows throughout
// the miner package.
package miner

import (
	"context"
	"crypto/sha256"
	"hash"

	"cosmossdk.io/depinject"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/either"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/observable/filter"
	"github.com/pokt-network/poktroll/pkg/observable/logging"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/protocol"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
)

var (
	_             relayer.Miner = (*miner)(nil)
	defaultHasher               = sha256.New()
	// TODO_BLOCKER: query on-chain governance params once available.
	// Setting this to 0 to effectively disable mining for now.
	// I.e., all relays are added to the tree.
	defaultRelayDifficulty = 0
)

// miner implements the relayer.Miner interface.
type miner struct {
	hasher          hash.Hash
	relayDifficulty int

	// Injected dependencies
	sessionManager relayer.RelayerSessionsManager
	blockClient    client.BlockClient
}

// NewMiner creates a new miner from the given dependencies and options. It
// returns an error if it has not been sufficiently configured or supplied.
func NewMiner(
	deps depinject.Config,
	opts ...relayer.MinerOption,
) (*miner, error) {
	mnr := &miner{}

	if err := depinject.Inject(
		deps,
		&mnr.sessionManager,
		&mnr.blockClient,
	); err != nil {
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

// TODO_IN_THIS_COMMIT: revisit comment...
// MinedRelays kicks off relay mining by mapping the servedRelays observable through
// a pipeline which hashes the relay, checks if it's above the mining difficulty,
// adds it to the session tree, and then maps any errors to a new observable.
// It also starts the claim and proof pipelines which are subsequently driven by
// mapping over RelayerSessionsManager's SessionsToClaim return observable.
// It does not block as map operations run in their own goroutines.
func (mnr *miner) MinedRelays(
	ctx context.Context,
	servedRelays observable.Observable[*servicetypes.Relay],
) observable.Observable[*relayer.MinedRelay] {
	// Map servedRelays observable to a new observable of an either type,
	// populated with the minedRelay or an error, which is notified after the
	// relay has been mined or an error has been encountered, respectively.
	eitherMinedRelays := channel.Map(ctx, servedRelays, mnr.mapMineRelay)
	logging.LogErrors(ctx, filter.EitherError(ctx, eitherMinedRelays))

	return filter.EitherSuccess(ctx, eitherMinedRelays)
}

// setDefaults ensures that the miner has been configured with a hasher and uses
// the default hasher if not.
func (mnr *miner) setDefaults() error {
	if mnr.hasher == nil {
		mnr.hasher = defaultHasher
	}
	return nil
}

// mapMineRelay is intended to be used as a MapFn. It hashes the relay and compares
// its difficulty to the minimum threshold. If the relay difficulty is sifficient,
// it returns an either populated with the MinedRelay value. Otherwise, it skips
// the relay. If it encounters an error, it returns an either populated with the
// error.
func (mnr *miner) mapMineRelay(
	_ context.Context,
	relay *servicetypes.Relay,
) (_ either.Either[*relayer.MinedRelay], skip bool) {
	relayBz, err := relay.Marshal()
	if err != nil {
		return either.Error[*relayer.MinedRelay](err), true
	}

	// Is it correct that we need to hash the key while smst.Update() could do it
	// since smst has a reference to the hasher
	mnr.hasher.Write(relayBz)
	relayHash := mnr.hasher.Sum(nil)
	mnr.hasher.Reset()

	if !protocol.BytesDifficultyGreaterThan(relayHash, defaultRelayDifficulty) {
		return either.Success[*relayer.MinedRelay](nil), true
	}

	return either.Success(&relayer.MinedRelay{
		Relay: *relay,
		Bytes: relayBz,
		Hash:  relayHash,
	}), false
}
