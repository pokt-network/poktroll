// NB: ensure this code is never included in any normal builds.
//go:build ignore

// NB: package MUST be `main` so that it can be run as a binary.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"hash"
	"log"
	"os"
	"sync"
	"time"

	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/miner"
	"github.com/pokt-network/poktroll/pkg/relayer/protocol"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
)

const (
	defaultDifficultyBits       = 16
	defaultFixtureLimitPerGroup = 5
	defaultRandLength           = 16
	defaultOutPath              = "relay_fixtures_test.go"
)

var (
	// flagDifficultyBitsThreshold is the number of leading zero bits that a
	// randomized, serialized relay must have to be included in the
	// `marshaledMinableRelaysHex` slice which is generated. It is also used as
	// the maximum difficulty allowed for relays to be included in the
	// `marshaledUnminableRelaysHex` slice.
	flagDifficultyBitsThreshold int

	// flagFixtureLimitPerGroup is the number of randomized, serialized relays that will be
	// generated for each of `marshaledMinableRelaysHex` and
	// `marshaledUnminableRelaysHex`.
	flagFixtureLimitPerGroup int

	// flagRandLength is the number of random bytes used for randomized values
	// during fixture generation.
	flagRandLength int

	// flagOut is the path to the generated file.
	flagOut string
)

// TODO_TECHDEBT: remove once marshaling using canonical codec.
type marshalable interface {
	Marshal() ([]byte, error)
}

func init() {
	flag.IntVar(&flagDifficultyBitsThreshold, "difficulty-bits-threshold", defaultDifficultyBits, "the number of leading zero bits that a randomized, serialized relay must have to be included in the `marshaledMinableRelaysHex` slice which is generated. It is also used as the maximum difficulty allowed for relays to be included in the `marshaledUnminableRelaysHex` slice.")
	flag.IntVar(&flagFixtureLimitPerGroup, "fixture-limit-per-group", defaultFixtureLimitPerGroup, "the number of randomized, serialized relays that will be generated for each of `marshaledMinableRelaysHex` and `marshaledUnminableRelaysHex`.")
	flag.IntVar(&flagRandLength, "rand-length", defaultRandLength, "the number of random bytes used for randomized values during fixture generation.")
	flag.StringVar(&flagOut, "out", defaultOutPath, "the path to the generated file.")
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
func main() {
	flag.Parse()

	const (
		randLength = 16 // number of random bytes provided for relay generation
	)
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	randRelaysObs, errCh := genRandomizedMinedRelayFixtures(
		ctx,
		flagRandLength,
		miner.DefaultRelayHasher,
	)
	exitOnError(errCh)

	outputBuffer := new(bytes.Buffer)

	// Append the beginning until the first `marshaledMinableRelaysHex` slice value.
	outputBuffer.WriteString(`package miner_test

var (
	// marshaledMinableRelaysHex are the hex encoded strings of serialized
	// relayer.MinedRelays which have been pre-mined to difficulty 2 by
	// populating the signature with random bytes. It is intended for use
	// in tests.
	marshaledMinableRelaysHex = []string{
`)

	// Append the minable relay fixtures.
	appendMinableRelays(ctx, randRelaysObs, outputBuffer)

	// Append the middle section between the two `marshaledMinableRelaysHex` and
	// `marshaledUnminableRelaysHex` slices.
	outputBuffer.WriteString(`	}

	// marshaledUnminableRelaysHex are the hex encoded strings of serialized
	// relayer.MinedRelays which have been pre-mined to **exclude** relays with
	// difficulty 2 (or greater). Like marshaledMinableRelaysHex, this is done
	// by populating the signature with random bytes. It is intended for use in
	// tests.
	marshaledUnminableRelaysHex = []string{
`)

	// Append the unminable relay fixtures.
	appendUnminableRelays(ctx, randRelaysObs, outputBuffer)

	// Append the end of the file.
	outputBuffer.WriteString(`	}
)
`)

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
	newHasher func() hash.Hash,
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
					Meta: &servicetypes.RelayRequestMetadata{
						Signature: randBz,
					},
					Payload: nil,
				},
				Res: nil,
			}

			// TODO_BLOCKER: use canonical codec.
			relayBz, err := relay.Marshal()
			if err != nil {
				errCh <- err
				return
			}

			// Hash relay bytes
			relayHash, err := hashBytes(newHasher, relayBz)
			if err != nil {
				errCh <- err
				return
			}

			randBzPublishCh <- &relayer.MinedRelay{
				Relay: relay,
				Bytes: relayBz,
				Hash:  relayHash,
			}
		}
	}()

	return randBzObs, errCh
}

// hashBytes hashes the given bytes using the given hasher.
func hashBytes(newHasher func() hash.Hash, relayBz []byte) ([]byte, error) {
	hasher := newHasher()
	if _, err := hasher.Write(relayBz); err != nil {
		return nil, err
	}

	return hasher.Sum(nil), nil
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

// appendMinableRelays maps over the given observable of mined relays, checks
// their difficulty against flagDifficultyBitsThreshold, and appends them to the
// given output buffer if it is greater than or equal to the threshold. It stops
// appending once flagFixtureLimitPerGroup number of relay fixtures have been
// appended.
func appendMinableRelays(
	ctx context.Context,
	randRelaysObs observable.Observable[*relayer.MinedRelay],
	outputBuffer *bytes.Buffer,
) {
	appendRelays(
		ctx,
		randRelaysObs,
		outputBuffer,
		func(hash []byte) bool {
			// Append if difficulty is greater than or equal to threshold.
			return protocol.MustCountDifficultyBits(hash) >= flagDifficultyBitsThreshold
		},
	)
}

// appendUnminableRelays maps over the given observable of mined relays, checks
// their difficulty against flagDifficultyBitsThreshold, and appends them to the
// given output buffer if it is less than the threshold. It stops appending once
// flagFixtureLimitPerGroup number of relay fixtures have been appended.
func appendUnminableRelays(
	ctx context.Context,
	randRelaysObs observable.Observable[*relayer.MinedRelay],
	outputBuffer *bytes.Buffer,
) {
	appendRelays(
		ctx,
		randRelaysObs,
		outputBuffer,
		func(hash []byte) bool {
			// Append if difficulty is less than threshold.
			return protocol.MustCountDifficultyBits(hash) < flagDifficultyBitsThreshold
		},
	)
}

// appendRelays maps over the given observable of mined relays, checks whether
// they should be appended to outpubBuffer by calling the given shouldAppend
// function, and appends them to the given output buffer if it returns true. It
// stops appending once flagFixtureLimitPerGroup number of relay fixtures have
// been appended.
func appendRelays(
	ctx context.Context,
	randRelaysObs observable.Observable[*relayer.MinedRelay],
	outputBuffer *bytes.Buffer,
	shouldAppend func(hash []byte) bool,
) {
	var (
		counterMu               sync.Mutex
		minedRelayAcceptCounter = 0
		minedRelayRejectCounter = 0
		mapCtx, cancelMap       = context.WithCancel(ctx)
	)

	filteredRelaysObs := channel.Map(mapCtx, randRelaysObs,
		func(
			_ context.Context,
			minedRelay *relayer.MinedRelay,
		) (_ *relayer.MinedRelay, skip bool) {
			counterMu.Lock()
			defer counterMu.Unlock()

			// At the end of each iteration, check if the relayCounter has reached
			// the limit. If so, cancel the mapCtx to stop the map operation.
			defer func() {
				if minedRelayAcceptCounter >= flagFixtureLimitPerGroup {
					cancelMap()
				}
			}()

			// Skip if shouldAppend returns false.
			if !shouldAppend(minedRelay.Hash) {
				minedRelayRejectCounter++
				return nil, true
			}

			// NB: slow down map loop, when not skipping, to prevent
			// overshooting the limit.
			time.Sleep(10 * time.Millisecond)

			minedRelayAcceptCounter++
			return minedRelay, false
		},
	)

	channel.ForEach(
		ctx, filteredRelaysObs,
		newAppendMarshalableHex[*relayer.MinedRelay](outputBuffer))

	// Wait for the map operation to finish.
	<-mapCtx.Done()
}

// newAppendMarshalableHex returns a new ForEachFn which appends the hex encoded
// string of the given marshalable to the given buffer.
func newAppendMarshalableHex[T marshalable](buf *bytes.Buffer) channel.ForEachFn[T] {
	return func(
		_ context.Context,
		marsh T,
	) {
		// TODO_BLOCKER: marshal using canonical codec.
		minedRelayBz, err := marsh.Marshal()
		if err != nil {
			log.Fatal(err)
		}

		if _, err := fmt.Fprintf(buf, "\t\t\"%x\",\n", minedRelayBz); err != nil {
			log.Fatal(err)
		}
	}
}
