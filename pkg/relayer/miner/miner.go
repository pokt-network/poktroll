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
	"log"

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
	supplierClient client.SupplierClient
	blockClient    client.BlockClient
}

// minedRelay is a wrapper around a relay that has been serialized and hashed.
type minedRelay struct {
	servicetypes.Relay
	bytes []byte
	hash  []byte
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
		&mnr.supplierClient,
		&mnr.blockClient,
	); err != nil {
		return nil, err
	}

	for _, opt := range opts {
		opt(mnr)
	}

	if err := mnr.validateConfigAndSetDefaults(); err != nil {
		return nil, err
	}

	return mnr, nil
}

// MineRelays kicks off relay mining by mapping the servedRelays observable through
// a pipeline which hashes the relay, checks if it's above the mining difficulty,
// adds it to the session tree, and then maps any errors to a new observable.
// It also starts the claim and proof pipelines which are subsequently driven by
// mapping over RelayerSessionsManager's SessionsToClaim return observable.
// It does not block as map operations run in their own goroutines.
func (mnr *miner) MineRelays(
	ctx context.Context,
	servedRelays observable.Observable[*servicetypes.Relay],
) {
	// sessiontypes.Relay ==> either.Either[minedRelay]
	eitherMinedRelays := channel.Map(ctx, servedRelays, mnr.mapMineRelay)

	// either.Either[minedRelay] ==> error
	miningErrors := channel.Map(ctx, eitherMinedRelays, mnr.mapAddRelayToSessionTree)
	logging.LogErrors(ctx, miningErrors)

	claimedSessions := mnr.createClaims(ctx)

	mnr.submitProofs(ctx, claimedSessions)
}

// createClaims maps over the RelaySessionsManager's SessionsToClaim return
// observable. For each claim, it calculates and waits for the earliest block
// height at which it is safe to claim and does so. It then maps any errors to
// a new observable which are subsequently logged. It returns an observable of
// the successfully claimed sessions. It does not block as map operations run
// in their own goroutines.
func (mnr *miner) createClaims(ctx context.Context) observable.Observable[relayer.SessionTree] {
	// Map SessionsToClaim observable to a new observable of the same type which
	// is notified when the session is eligible to be claimed.
	// relayer.SessionTree ==> relayer.SessionTree
	sessionsWithOpenClaimWindow := channel.Map(
		ctx, mnr.sessionManager.SessionsToClaim(),
		mnr.mapWaitForEarliestCreateClaimHeight,
	)

	failedCreateClaimSessions, failedCreateClaimSessionsPublishCh :=
		channel.NewObservable[relayer.SessionTree]()

	// Map sessionsWithOpenClaimWindow to a new observable of an either type,
	// populated with the session or an error, which is notified after the session
	// claim has been created or an error has been encountered, respectively.
	eitherClaimedSessions := channel.Map(
		ctx, sessionsWithOpenClaimWindow,
		mnr.newMapClaimSessionFn(failedCreateClaimSessionsPublishCh),
	)

	// TODO_TECHDEBT: pass failed create claim sessions to some retry mechanism.
	_ = failedCreateClaimSessions
	logging.LogErrors(ctx, filter.EitherError(ctx, eitherClaimedSessions))

	// Map eitherClaimedSessions to a new observable of relayer.SessionTree which
	// is notified when the corresponding claim creation succeeded.
	return filter.EitherSuccess(ctx, eitherClaimedSessions)
}

// submitProofs maps over the given claimedSessions observable. For each session,
// it calculates and waits for the earliest block height at which it is safe to
// submit a proof and does so. It then maps any errors to a new observable which
// are subsequently logged. It does not block as map operations run in their own
// goroutines.
func (mnr *miner) submitProofs(
	ctx context.Context,
	claimedSessions observable.Observable[relayer.SessionTree],
) {
	// Map claimedSessions to a new observable of the same type which is notified
	// when the session is eligible to be proven.
	sessionsWithOpenProofWindow := channel.Map(
		ctx, claimedSessions,
		mnr.mapWaitForEarliestSubmitProofHeight,
	)

	failedSubmitProofSessions, failedSubmitProveSessionsPublishCh :=
		channel.NewObservable[relayer.SessionTree]()

	// Map sessionsWithOpenProofWindow to a new observable of an either type,
	// populated with the session or an error, which is notified after the session
	// proof has been submitted or an error has been encountered, respectively.
	eitherProvenSessions := channel.Map(
		ctx, sessionsWithOpenProofWindow,
		mnr.newMapProveSessionFn(failedSubmitProveSessionsPublishCh),
	)

	// TODO_TECHDEBT: pass failed submit proof sessions to some retry mechanism.
	_ = failedSubmitProofSessions
	logging.LogErrors(ctx, filter.EitherError(ctx, eitherProvenSessions))
}

// validateConfigAndSetDefaults ensures that the miner has been configured with
// a hasher and uses the default hasher if not.
func (mnr *miner) validateConfigAndSetDefaults() error {
	if mnr.hasher == nil {
		mnr.hasher = defaultHasher
	}
	return nil
}

// mapMineRelay is intended to be used as a MapFn. It hashes the relay and compares
// its difficulty to the minimum threshold. If the relay difficulty is sifficient,
// it returns an either populated with the minedRelay value. Otherwise, it skips
// the relay. If it encounters an error, it returns an either populated with the
// error.
func (mnr *miner) mapMineRelay(
	_ context.Context,
	relay *servicetypes.Relay,
) (_ either.Either[*minedRelay], skip bool) {
	relayBz, err := relay.Marshal()
	if err != nil {
		return either.Error[*minedRelay](err), true
	}

	// Is it correct that we need to hash the key while smst.Update() could do it
	// since smst has a reference to the hasher
	mnr.hasher.Write(relayBz)
	relayHash := mnr.hasher.Sum(nil)
	mnr.hasher.Reset()

	if !protocol.BytesDifficultyGreaterThan(relayHash, defaultRelayDifficulty) {
		return either.Success[*minedRelay](nil), true
	}

	return either.Success(&minedRelay{
		Relay: *relay,
		bytes: relayBz,
		hash:  relayHash,
	}), false
}

// mapAddRelayToSessionTree is intended to be used as a MapFn. It adds the relay
// to the session tree. If it encounters an error, it returns the error. Otherwise,
// it skips output (only outputs errors).
func (mnr *miner) mapAddRelayToSessionTree(
	_ context.Context,
	eitherRelay either.Either[*minedRelay],
) (_ error, skip bool) {
	// Propagate any upstream errors.
	relay, err := eitherRelay.ValueOrError()
	if err != nil {
		return err, false
	}

	// ensure the session tree exists for this relay
	sessionHeader := relay.GetReq().GetMeta().GetSessionHeader()
	smst, err := mnr.sessionManager.EnsureSessionTree(sessionHeader)
	if err != nil {
		log.Printf("failed to ensure session tree: %s\n", err)
		return err, false
	}

	if err := smst.Update(relay.hash, relay.bytes, 1); err != nil {
		log.Printf("failed to update smt: %s\n", err)
		return err, false
	}

	// Skip because this map function only outputs errors.
	return nil, true
}

// mapWaitForEarliestCreateClaimHeight is intended to be used as a MapFn. It
// calculates and waits for the earliest block height, allowed by the protocol,
// at which a claim can be created for the given session, then emits the session
// **at that moment**.
func (mnr *miner) mapWaitForEarliestCreateClaimHeight(
	ctx context.Context,
	session relayer.SessionTree,
) (_ relayer.SessionTree, skip bool) {
	mnr.waitForEarliestCreateClaimHeight(
		ctx, session.GetSessionHeader().GetSessionEndBlockHeight(),
	)
	return session, false
}

// waitForEarliestCreateClaimHeight calculates and waits for the earliest block
// height, allowed by the protocol, at which a claim can be created for a session
// with the given sessionEndHeight. It is calculated relative to sessionEndHeight
// using on-chain governance parameters and randomized input.
func (mnr *miner) waitForEarliestCreateClaimHeight(
	ctx context.Context,
	sessionEndHeight int64,
) {
	// TODO_TECHDEBT: refactor this logic to a shared package.

	createClaimWindowStartHeight := sessionEndHeight
	// TODO_TECHDEBT: query the on-chain governance parameter once available.
	// + claimproofparams.GovCreateClaimWindowStartHeightOffset

	// we wait for createClaimWindowStartHeight to be received before proceeding since we need its hash
	// to know where this servicer's claim submission window starts.
	log.Printf("waiting for global earliest claim submission createClaimWindowStartBlock height: %d", createClaimWindowStartHeight)
	createClaimWindowStartBlock := mnr.waitForBlock(ctx, createClaimWindowStartHeight)

	log.Printf("received earliest claim submission createClaimWindowStartBlock height: %d, use its hash to have a random submission for the servicer", createClaimWindowStartBlock.Height())

	earliestCreateClaimHeight :=
		protocol.GetEarliestCreateClaimHeight(createClaimWindowStartBlock)

	log.Printf("earliest claim submission createClaimWindowStartBlock height for this supplier: %d", earliestCreateClaimHeight)
	_ = mnr.waitForBlock(ctx, earliestCreateClaimHeight)
}

// waitForBlock blocks until the block at the given height (or greater) is
// observed as having been committed.
func (mnr *miner) waitForBlock(ctx context.Context, height int64) client.Block {
	subscription := mnr.blockClient.CommittedBlocksSequence(ctx).Subscribe(ctx)
	defer subscription.Unsubscribe()

	for block := range subscription.Ch() {
		if block.Height() >= height {
			return block
		}
	}

	return nil
}

// newMapClaimSessionFn returns a new MapFn that creates a claim for the given
// session. Any session which encouters an error while creating a claim is sent
// on the failedCreateClaimSessions channel.
func (mnr *miner) newMapClaimSessionFn(
	failedCreateClaimSessions chan<- relayer.SessionTree,
) channel.MapFn[relayer.SessionTree, either.SessionTree] {
	return func(
		ctx context.Context,
		session relayer.SessionTree,
	) (_ either.SessionTree, skip bool) {
		// this session should no longer be updated
		claimRoot, err := session.Flush()
		if err != nil {
			return either.Error[relayer.SessionTree](err), false
		}

		sessionHeader := session.GetSessionHeader()
		if err := mnr.supplierClient.CreateClaim(ctx, *sessionHeader, claimRoot); err != nil {
			failedCreateClaimSessions <- session
			return either.Error[relayer.SessionTree](err), false
		}

		return either.Success(session), false
	}
}

// mapWaitForEarliestSubmitProofHeight is intended to be used as a MapFn. It
// calculates and waits for the earliest block height, allowed by the protocol,
// at which a proof can be submitted for the given session, then emits the session
// **at that moment**.
func (mnr *miner) mapWaitForEarliestSubmitProofHeight(
	ctx context.Context,
	session relayer.SessionTree,
) (_ relayer.SessionTree, skip bool) {
	mnr.waitForEarliestSubmitProofHeight(
		ctx, session.GetSessionHeader().GetSessionEndBlockHeight(),
	)
	return session, false
}

// waitForEarliestSubmitProofHeight calculates and waits for the earliest block
// height, allowed by the protocol, at which a proof can be submitted for a session
// which was claimed at createClaimHeight. It is calculated relative to
// createClaimHeight using on-chain governance parameters and randomized input.
func (mnr *miner) waitForEarliestSubmitProofHeight(
	ctx context.Context,
	createClaimHeight int64,
) {
	submitProofWindowStartHeight := createClaimHeight
	// TODO_TECHDEBT: query the on-chain governance parameter once available.
	// + claimproofparams.GovSubmitProofWindowStartHeightOffset

	// we wait for submitProofWindowStartHeight to be received before proceeding since we need its hash
	log.Printf("waiting for global earliest proof submission submitProofWindowStartBlock height: %d", submitProofWindowStartHeight)
	submitProofWindowStartBlock := mnr.waitForBlock(ctx, submitProofWindowStartHeight)

	earliestSubmitProofHeight := protocol.GetEarliestSubmitProofHeight(submitProofWindowStartBlock)
	_ = mnr.waitForBlock(ctx, earliestSubmitProofHeight)
}

// newMapProveSessionFn returns a new MapFn that submits a proof for the given
// session. Any session which encouters errors while submitting a proof is sent
// on the failedSubmitProofSessions channel.
func (mnr *miner) newMapProveSessionFn(
	failedSubmitProofSessions chan<- relayer.SessionTree,
) channel.MapFn[relayer.SessionTree, either.SessionTree] {
	return func(
		ctx context.Context,
		session relayer.SessionTree,
	) (_ either.SessionTree, skip bool) {
		latestBlock := mnr.blockClient.LatestBlock(ctx)
		proof, err := session.ProveClosest(latestBlock.Hash())
		if err != nil {
			return either.Error[relayer.SessionTree](err), false
		}

		log.Printf("currentBlock: %d, submitting proof", latestBlock.Height()+1)
		// SubmitProof ensures on-chain proof inclusion so we can safely prune the tree.
		if err := mnr.supplierClient.SubmitProof(
			ctx,
			*session.GetSessionHeader(),
			proof,
		); err != nil {
			failedSubmitProofSessions <- session
			return either.Error[relayer.SessionTree](err), false
		}

		return either.Success(session), false
	}
}
