// NB: ensure this code is never included in any normal builds.
//go:build ignore

// NB: package MUST be `main` so that it can be run as a binary.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/testutil/sample"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const (
	// defaultDifficultyThresholdHashStr is the default difficulty hash as a hex string, used to separate the generated relays into minable and unminable categories.
	// This hash should match the service-specific difficulty target hash used by the relay miner tests in pkg/relayer/miner/miner_test.go.
	defaultDifficultyThresholdHashStr = "0000ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	defaultFixtureLimitPerGroup       = 5
	defaultRandLength                 = 16
	defaultOutPath                    = "relay_fixtures_test.go"

	// defaultSvcID is the ID of the service referenced in each relay request's metadata.
	// This is passed to the tokenomics client to get the service's difficulty target hash.
	// It should match the testSvcId const in pkg/relayer/miner/miner_test.go
	defaultSvcID = "svc1"
)

// TODO_FOLLOWUP(@olshansk, #690): Do a global anycase grep for "DifficultyBits" and update/remove things appropriately.
var (
	// flagDifficultyThresholdHashStr is the difficulty threshold hash, as a hex string, that a
	// randomized, serialized relay must be greater than to be included in the
	// generated `marshaledMinableRelaysHex` slice.
	// It is also used as the maximum difficulty threshold hash, below which, relays
	// included in the generated `marshaledUnminableRelaysHex` slice.
	flagDifficultyThresholdHashStr string

	// flagFixtureLimitPerGroup is the number of randomized, serialized relays that will be
	// generated for each of `marshaledMinableRelaysHex` and
	// `marshaledUnminableRelaysHex`.
	flagFixtureLimitPerGroup int

	// flagOut is the path to the generated file.
	flagOut string

	// flagSvcID is the service ID referenced in generated relay fixtures.
	flagSvcID string
)

// TODO_TECHDEBT: remove once marshaling using canonical codec.
type marshalable interface {
	Marshal() ([]byte, error)
}

func init() {
	flag.StringVar(&flagDifficultyThresholdHashStr, "difficulty-threshold-hash-str", defaultDifficultyThresholdHashStr, "the difficulty threshold hash, in string format, that a randomized, serialized relay be greater than or equal to when hashed, to be included in the `marshaledMinableRelaysHex` slice which is generated. It is also used as the maximum difficulty threshold hash for relays to be included in the `marshaledUnminableRelaysHex` slice.")
	flag.IntVar(&flagFixtureLimitPerGroup, "fixture-limit-per-group", defaultFixtureLimitPerGroup, "the number of randomized, serialized relays that will be generated for each of `marshaledMinableRelaysHex` and `marshaledUnminableRelaysHex`.")
	flag.StringVar(&flagOut, "out", defaultOutPath, "the path to the generated file.")
	flag.StringVar(&flagSvcID, "service-id", defaultSvcID, "the service ID referenced in generated relays.")
}

// This is utility for generating relay fixtures for testing. It is not intended
// to be used **in/by** any tests but rather is persisted to aid in re-generation
// of relay fixtures should the test requirements change. It generates two slices
// of minedRelays, `marshaledMinableRelaysHex` and `marshaledUnminableRelaysHex`,
// which contain hex encoded strings of serialized relays. The relays in
// `marshaledMinableRelaysHex` have been pre-mined to difficulty 16 by populating
// the signature with random bytes. The relays in `marshaledUnminableRelaysHex`
// have been pre-mined to **exclude** relays with difficulty 16 (or greater). Like
// `marshaledMinableRelaysHex`, this is done by populating the signature with
// random bytes.
// Output file is truncated and overwritten if it already exists.
//
// To regenerate all fixtures, use `make go_testgen_fixtures`; to regenerate only this
// test's fixtures run `go generate ./pkg/relayer/miner/miner_test.go`.
func main() {
	flag.Parse()

	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	randRelaysObs, errCh := genRandomizedMinedRelayFixtures(
		ctx,
		defaultRandLength,
	)
	exitOnError(errCh)

	outputBuffer := new(bytes.Buffer)

	// Collect the minable relay fixtures into a single string (one relay per line).
	marshaledMinableRelaysHex := getMarshaledRelayFmtLines(ctx, randRelaysObs, difficultyGTE)

	// Collect the unminable relay fixtures into a single string (one relay per line).
	marshaledUnminableRelaysHex := getMarshaledRelayFmtLines(ctx, randRelaysObs, difficultyLT)

	// Interpolate the collected relay fixtures into the relay fixtures template.
	if err := relayFixturesTemplate.Execute(
		outputBuffer,
		map[string]any{
			"difficultyThresholdHashStr":  flagDifficultyThresholdHashStr,
			"MarshaledMinableRelaysHex":   marshaledMinableRelaysHex,
			"MarshaledUnminableRelaysHex": marshaledUnminableRelaysHex,
		},
	); err != nil {
		log.Fatal(err)
	}

	// Write the output buffer to the file at flagOut path.
	if err := os.WriteFile(flagOut, outputBuffer.Bytes(), 0644); err != nil {
		log.Fatal(err)
	}
}

// genRandomizedMinedRelayFixtures returns an observable of mined relays which
// are generated by randomizing the signature of a relay. It generates these mined
// relay fixtures continuously until the context is canceled. It also returns an
// error channel which will receive any error it encounters while generating.
func genRandomizedMinedRelayFixtures(
	ctx context.Context,
	randLength int,
) (observable.Observable[*relayer.MinedRelay], <-chan error) {
	var (
		errCh                      = make(chan error, 1)
		randBzObs, randBzPublishCh = channel.NewObservable[*relayer.MinedRelay]()
	)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			randBz := make([]byte, randLength)
			if _, err := rand.Read(randBz); err != nil {
				errCh <- err
				return
			}

			// Populate a relay with the minimally sufficient randomized data.
			relay := servicetypes.Relay{
				Req: &servicetypes.RelayRequest{
					Meta: servicetypes.RelayRequestMetadata{
						SessionHeader: &sessiontypes.SessionHeader{
							ApplicationAddress: sample.AccAddress(),
							Service: &sharedtypes.Service{
								Id: flagSvcID,
							},
							SessionId:               "session_id",
							SessionStartBlockHeight: 1,
							SessionEndBlockHeight:   2,
						},

						Signature: randBz,
					},
					Payload: nil,
				},
				Res: nil,
			}

			relayBz, err := relay.Marshal()
			if err != nil {
				errCh <- err
				return
			}

			relayHashArr := protocol.GetRelayHashFromBytes(relayBz)

			randBzPublishCh <- &relayer.MinedRelay{
				Relay: relay,
				Bytes: relayBz,
				Hash:  relayHashArr[:],
			}
		}
	}()

	return randBzObs, errCh
}

// exitOnError exits the program if an error is received on the given error
// channel.
func exitOnError(errCh <-chan error) {
	go func() {
		for err := range errCh {
			log.Fatalf("ERROR: %s", err)
		}
	}()
}

// getDifficultyThresholdHash returns the difficulty threshold hash represented by the
// flagDifficultyThresholdHashStr flag, or the default value if the flag has not been set.
func getDifficultyThresholdHash() []byte {
	difficultyThresholdHash, _ := hex.DecodeString(flagDifficultyThresholdHashStr)
	return difficultyThresholdHash
}

// difficultyGTE returns true if the given hash has a difficulty greater than or
// equal to that represented by flagDifficultyThresholdHashStr.
func difficultyGTE(hash []byte) bool {
	return protocol.IsRelayVolumeApplicable(hash, getDifficultyThresholdHash())
}

// difficultyLT returns true if the given hash has a difficulty less than
// that represented by flagDifficultyThresholdHashStr.
func difficultyLT(hash []byte) bool {
	return !protocol.IsRelayVolumeApplicable(hash, getDifficultyThresholdHash())
}

// TODO_IMPROVE: this function could be simplified to a great extent to make it easier to understand the
// test cases.
// getMarshaledRelayFmtLines performs two map operations followed by a collect.
// The first map filters mined relays from the given observable, skipping when
// shouldAccept is false. This map, and as a result, all downstream observables
// are closed when flagFixtureLimitPerGroup number of relays have been accepted.
// The second map then marshals, hex-encodes, and formats the filtered mined relay.
// Finally, the collect operation collects the formatted mined relays into a slice
// to return.
func getMarshaledRelayFmtLines(
	ctx context.Context,
	randRelaysObs observable.Observable[*relayer.MinedRelay],
	shouldAccept func(hash []byte) bool,
) string {
	ctx, cancelFilterMapCollect := context.WithCancel(ctx)
	filteredRelaysObs := filterLimitRelays(
		ctx,
		cancelFilterMapCollect,
		flagFixtureLimitPerGroup,
		randRelaysObs,
		shouldAccept,
	)

	marshaledFilteredRelayLinesObs := channel.Map(
		ctx, filteredRelaysObs,
		newMapRelayMarshalLineFmt[*relayer.MinedRelay](relayFixtureLineFmt),
	)

	// Collect the filtered relays and return them (as a slice).
	marshaledFilteredRelayLines := channel.Collect(ctx, marshaledFilteredRelayLinesObs)
	return strings.Join(marshaledFilteredRelayLines, "\n")
}

// filterLimitRelays maps over the given observable of mined relays, skipping when
// the given shouldAppend function returns false. Once flagFixtureLimitPerGroup
// number of relay fixtures have been mapped, it calls the given cancel function.
func filterLimitRelays(
	ctx context.Context,
	cancel context.CancelFunc,
	limit int,
	randRelaysObs observable.Observable[*relayer.MinedRelay],
	shouldCollect func(hash []byte) bool,
) observable.Observable[*relayer.MinedRelay] {
	var (
		counterMu               sync.Mutex
		minedRelayAcceptCounter = 0
		minedRelayRejectCounter = 0
	)

	return channel.Map(ctx, randRelaysObs,
		func(
			_ context.Context,
			minedRelay *relayer.MinedRelay,
		) (_ *relayer.MinedRelay, skip bool) {
			counterMu.Lock()
			defer counterMu.Unlock()

			// At the start of each iteration, check if the relayCounter has reached
			// the limit. If so, cancel the ctx to stop the map operation.
			if minedRelayAcceptCounter >= limit {
				// Wait a tick for the map to complete as the observable drains
				// asynchronously.
				time.Sleep(time.Millisecond)
				cancel()
				return nil, true
			}

			// Skip if shouldCollect returns false.
			if !shouldCollect(minedRelay.Hash) {
				minedRelayRejectCounter++
				return nil, true
			}

			minedRelayAcceptCounter++
			return minedRelay, false
		},
	)
}

// newMapRelayMarshalLineFmt returns a MapFn which formats the given marshalable
// as a hex-encoded string with the given line format string.
func newMapRelayMarshalLineFmt[T marshalable](lineFmt string) channel.MapFn[T, string] {
	return func(
		_ context.Context,
		marsh T,
	) (_ string, skip bool) {
		// TODO_TECHDEBT(@red-0ne): marshal using canonical codec.
		minedRelayBz, err := marsh.Marshal()
		if err != nil {
			log.Fatal(err)
		}

		return fmt.Sprintf(lineFmt, minedRelayBz), false
	}
}
