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

// TODO: https://stackoverflow.com/questions/77190071/golang-best-practice-for-functions-intended-to-be-called-as-goroutines

// TODO_THIS_COMMIT: define interface & unexport
// TODO_COMMENT: Explain what the responsibility of this structure is, how its used throughout
// and leave comments alongside each field.
type miner struct {
	hasher          hash.Hash
	relayDifficulty int

	// Injected dependencies
	sessionManager relayer.RelayerSessionsManager
	supplierClient client.SupplierClient
	blockClient    client.BlockClient
}

type minedRelay struct {
	servicetypes.Relay
	bytes []byte
	hash  []byte
}

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

// MineRelays assigns the servedRelays and sessions observables & starts their respective consumer goroutines.
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

func (mnr *miner) validateConfigAndSetDefaults() error {
	if mnr.hasher == nil {
		mnr.hasher = defaultHasher
	}
	return nil
}

func (mnr *miner) mineRelays(
	ctx context.Context,
	servedRelays observable.Observable[servicetypes.Relay],
) observable.Observable[either.Either[minedRelay]] {
	// servicetypes.Relay ==> either.Either[minedRelay]
	return channel.Map(ctx, servedRelays, mnr.mapMineRelay)
}

// TODO_THIS_COMMIT: update comment.
// mapMineRelay validates, executes, & hashes the relay. If the relay's difficulty
// is above the mining difficulty, it's inserted into SMST.
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

func (mnr *miner) addReplayToSessionTree(
	ctx context.Context,
	eitherMinedRelays observable.Observable[either.Either[minedRelay]],
) observable.Observable[error] {
	// either.Either[minedRelay] ==> error
	return channel.Map(ctx, eitherMinedRelays, mnr.mapAddRelayToSessionTree)
}

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

func (mnr *miner) waitForOpenClaimWindow(
	ctx context.Context,
	sessionsToClaim observable.Observable[relayer.SessionTree],
) observable.Observable[relayer.SessionTree] {
	// relayer.SessionTree ==> relayer.SessionTree
	return channel.Map(ctx, sessionsToClaim, mnr.mapWaitForOpenClaimWindow)
}

func (mnr *miner) mapWaitForOpenClaimWindow(
	ctx context.Context,
	session relayer.SessionTree,
) (_ relayer.SessionTree, skip bool) {
	mnr.waitForEarliestCreateClaimDistributionHeight(
		ctx, session.GetSessionHeader().GetSessionEndBlockHeight(),
	)

	// TODO_THIS_COMMIT: reconsider logging...
	//log.Printf("currentBlock: %d, creating claim", block.Height())
	return session, false
}

// waitForEarliestCreateClaimDistributionHeight returns the earliest and latest block heights at which
// a claim can be submitted for the current session.
// explanation of how the earliest and latest submission block height is determined is available in the
// poktroll/x/servicer/keeper/msg_server_claim.go file
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
// observed as committed.
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

func (mnr *miner) mapWaitForOpenProofWindow(
	ctx context.Context,
	session relayer.SessionTree,
) (_ relayer.SessionTree, skip bool) {
	mnr.waitForEarliestSubmitProofDistributionHeight(
		ctx, session.GetSessionHeader().GetSessionEndBlockHeight(),
	)

	// TODO_THIS_COMMIT: reconsider logging...
	//log.Printf("currentBlock: %d, submitting proof", block.Height())
	return session, false
}

// getProofSubmissionWindow returns the earliest and latest block heights at which
// a proof can be submitted.
// explanation of how the earliest and latest submission block height is determined is available in the
// poktroll/x/servicer/keeper/msg_server_claim.go file
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
