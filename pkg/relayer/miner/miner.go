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
	// TODO_THIS_COMMIT: where (on-chain) should this come from?
	// TODO_TECHDEBT: setting this to 0 to effectively disable mining for now.
	// I.e. all relays are added to the tree.
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
// adds it to the session tree, and then maps any errors to a  new observable.
// It also starts the claim and proof pipelines which are subsequently driven by
// mapping over RelayerSessionsManager's SessionsToClaim return observable.
// It does not block as map operations run in their own goroutines.
func (mnr *miner) MineRelays(
	ctx context.Context,
	servedRelays observable.Observable[servicetypes.Relay],
) {
	// sessiontypes.Relay ==> either.Either[minedRelay]
	eitherMinedRelays := mnr.mineRelays(ctx, servedRelays)

	// either.Either[minedRelay] ==> error
	miningErrors := mnr.addReplayToSessionTree(ctx, eitherMinedRelays)
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
	// relayer.SessionTree ==> either.SessionTree
	sessionsWithOpenClaimWindow := mnr.waitForOpenClaimWindow(ctx, mnr.sessionManager.SessionsToClaim())

	failedCreateClaimSessions, failedCreateClaimSessionsPublishCh :=
		channel.NewObservable[relayer.SessionTree]()

	// either.SessionTree ==> either.SessionTree
	eitherClaimedSessions := channel.Map(
		ctx, sessionsWithOpenClaimWindow,
		mnr.newMapClaimSessionFn(failedCreateClaimSessionsPublishCh),
	)

	// TODO_TECHDEBT: pass failed create claim sessions to some retry mechanism.
	_ = failedCreateClaimSessions
	logging.LogErrors(ctx, filter.EitherError(ctx, eitherClaimedSessions))

	// either.SessionTree ==> relayer.SessionTree
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
	// relayer.SessionTree ==> relayer.SessionTree
	sessionsWithOpenProofWindow := channel.Map(ctx, claimedSessions, mnr.mapWaitForOpenProofWindow)

	failedSubmitProofSessions, failedSubmitProveSessionsPublishCh := channel.NewObservable[relayer.SessionTree]()

	// relayer.SessionTree ==> relayer.SessionTree
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

// mineRelays maps over the servedRelays observable, applyging the mapMineRelay
// method to each relay. It returns an observable of the mined relays.
func (mnr *miner) mineRelays(
	ctx context.Context,
	servedRelays observable.Observable[servicetypes.Relay],
) observable.Observable[either.Either[minedRelay]] {
	// servicetypes.Relay ==> either.Either[minedRelay]
	return channel.Map(ctx, servedRelays, mnr.mapMineRelay)
}

// mapMineRelay is intended to be used as a MapFn. It hashes the relay and compares
// its difficulty to the minimum threshold. If the relay difficulty is sifficient,
// it returns an either populated with the minedRelay value. Otherwise, it skips
// the relay. If it encounters an error, it returns an either populated with the
// error.
func (mnr *miner) mapMineRelay(
	_ context.Context,
	relay servicetypes.Relay,
) (_ either.Either[minedRelay], skip bool) {
	relayBz, err := relay.Marshal()
	if err != nil {
		return either.Error[minedRelay](err), true
	}

	// Is it correct that we need to hash the key while smst.Update() could do it
	// since smst has a reference to the hasher
	mnr.hasher.Write(relayBz)
	relayHash := mnr.hasher.Sum(nil)
	mnr.hasher.Reset()

	if !protocol.BytesDifficultyGreaterThan(relayHash, defaultRelayDifficulty) {
		return either.Success(minedRelay{}), true
	}

	return either.Success(minedRelay{
		Relay: relay,
		bytes: relayBz,
		hash:  relayHash,
	}), false
}

// addReplayToSessionTree maps over the eitherMinedRelays observable, applying the
// mapAddRelayToSessionTree method to each relay. It returns an observable of the
// errors encountered.
func (mnr *miner) addReplayToSessionTree(
	ctx context.Context,
	eitherMinedRelays observable.Observable[either.Either[minedRelay]],
) observable.Observable[error] {
	// either.Either[minedRelay] ==> error
	return channel.Map(ctx, eitherMinedRelays, mnr.mapAddRelayToSessionTree)
}

// mapAddRelayToSessionTree is intended to be used as a MapFn. It adds the relay
// to the session tree. If it encounters an error, it returns the error. Otherwise,
// it skips output (only outputs errors).
func (mnr *miner) mapAddRelayToSessionTree(
	_ context.Context,
	eitherRelay either.Either[minedRelay],
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

// waitForOpenClaimWindow maps over the SessionsToClaim observable, applying the
// mapWaitForOpenClaimWindow method to each session. It returns an observable of
// the sessions that are ready to be claimed which is notified when the respective
// session is ready to be claimed.
func (mnr *miner) waitForOpenClaimWindow(
	ctx context.Context,
	sessionsToClaim observable.Observable[relayer.SessionTree],
) observable.Observable[relayer.SessionTree] {
	// relayer.SessionTree ==> relayer.SessionTree
	return channel.Map(ctx, sessionsToClaim, mnr.mapWaitForOpenClaimWindow)
}

// mapWaitForOpenClaimWindow is intended to be used as a MapFn. It calculates and
// waits for the earliest block height at which it is safe to claim and returns
// the session when it should be claimed.
func (mnr *miner) mapWaitForOpenClaimWindow(
	ctx context.Context,
	session relayer.SessionTree,
) (_ relayer.SessionTree, skip bool) {
	mnr.waitForEarliestCreateClaimDistributionHeight(
		ctx, session.GetSessionHeader().GetSessionEndBlockHeight(),
	)
	return session, false
}

// waitForEarliestCreateClaimDistributionHeight returns the earliest block height
// at which a claim can be submitted. It is calculated from the session end block
// height, on-chain governance parameters, and randomized input.
func (mnr *miner) waitForEarliestCreateClaimDistributionHeight(
	ctx context.Context,
	sessionEndHeight int64,
) {
	// TODO_TECHDEBT: refactor this logic to a shared package.

	earliestCreateClaimBlockHeight := sessionEndHeight
	// TODO_TECHDEBT: query the on-chain governance parameter once available.
	// + claimproofparams.GovEarliestClaimSubmissionBlocksOffset

	// we wait for earliestCreateClaimBlockHeight to be received before proceeding since we need its hash
	// to know where this servicer's claim submission window starts.
	log.Printf("waiting for global earliest claim submission earliestCreateClaimBlock height: %d", earliestCreateClaimBlockHeight)
	earliestCreateClaimBlock := mnr.waitForBlock(ctx, earliestCreateClaimBlockHeight)

	log.Printf("received earliest claim submission earliestCreateClaimBlock height: %d, use its hash to have a random submission for the servicer", earliestCreateClaimBlock.Height())

	earliestClaimSubmissionDistributionHeight := protocol.GetCreateClaimDistributionHeight(earliestCreateClaimBlock)

	log.Printf("earliest claim submission earliestCreateClaimBlock height for this supplier: %d", earliestClaimSubmissionDistributionHeight)
	_ = mnr.waitForBlock(ctx, earliestClaimSubmissionDistributionHeight)

	// TODO_THIS_COMMIT: this didn't seem to be used, confirm and remove.
	// TODO_TECHDEBT: query the on-chain governance parameter once available.
	// latestServicerClaimSubmissionBlockHeight := earliestClaimSubmissionDistributionHeight +
	//   claimproofparams.GovClaimSubmissionBlocksWindow + 1
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
// session. Any session which encouters errors while creating a claim is sent
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
			// TODO_THIS_COMMIT: cleanup error handling/logging
			log.Printf("failed to close tree: %s", err)
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

// mapWaitForOpenProofWindow maps over the claimedSessions observable, applying
// the mapWaitForOpenProofWindow method to each session. It returns an observable
// of the sessions that are ready to be proven which is notified when the respective
// session is ready to be proven.
func (mnr *miner) mapWaitForOpenProofWindow(
	ctx context.Context,
	session relayer.SessionTree,
) (_ relayer.SessionTree, skip bool) {
	mnr.waitForEarliestSubmitProofDistributionHeight(
		ctx, session.GetSessionHeader().GetSessionEndBlockHeight(),
	)
	return session, false
}

// waitForEarliestSubmitProofDistributionHeight returns the earliest block height
// at which a proof can be submitted. It is calculated from the session claim
// creation block height, on-chain governance parameters, and randomized input.
func (mnr *miner) waitForEarliestSubmitProofDistributionHeight(
	ctx context.Context,
	createClaimHeight int64,
) {
	earliestSubmitProofBlockHeight := createClaimHeight
	// TODO_TECHDEBT: query the on-chain governance parameter once available.
	// + claimproofparams.GovEarliestProofSubmissionBlocksOffset

	// we wait for earliestSubmitProofBlockHeight to be received before proceeding since we need its hash
	log.Printf("waiting for global earliest proof submission earliestSubmitProofBlock height: %d", earliestSubmitProofBlockHeight)
	earliestSubmitProofBlock := mnr.waitForBlock(ctx, earliestSubmitProofBlockHeight)

	earliestSubmitProofDistributionHeight := protocol.GetSubmitProofDistributionHeight(earliestSubmitProofBlock)
	_ = mnr.waitForBlock(ctx, earliestSubmitProofDistributionHeight)

	// TODO_THIS_COMMIT: this didn't seem to be used, confirm and remove.
	// TODO_TECHDEBT: query the on-chain governance parameter once available.
	// latestServicerClaimSubmissionBlockHeight := earliestSubmitProofBlockHeight +
	//   claimproofparams.GovProofSubmissionBlocksWindow + 1
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
