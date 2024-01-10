package channel_test

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/testutil/testchannel"
	"github.com/pokt-network/poktroll/testutil/testerrors"
)

const (
	publishDelay           = time.Millisecond
	notifyTimeout          = 50 * time.Millisecond
	cancelUnsubscribeDelay = publishDelay * 2
)

func TestChannelObservable_NotifyObservers(t *testing.T) {
	type test struct {
		name            string
		publishCh       chan int
		inputs          []int
		expectedOutputs []int
		setupFn         func(t test)
	}

	inputs := []int{123, 456, 789}
	// NB: see TODO_INCOMPLETE comment below
	// fullBlockingPublisher := make(chan *int)
	// fullBufferedPublisher := make(chan *int, 1)

	tests := []test{
		{
			name:            "nil publisher (default buffer size)",
			publishCh:       nil,
			inputs:          inputs,
			expectedOutputs: inputs,
		},
		{
			name:            "empty non-buffered publisher",
			publishCh:       make(chan int),
			inputs:          inputs,
			expectedOutputs: inputs,
		},
		{
			name:            "empty buffered len 1 publisher",
			publishCh:       make(chan int, 1),
			inputs:          inputs,
			expectedOutputs: inputs,
		},
		{
			name:            "empty buffered len 1000 publisher",
			publishCh:       make(chan int, 1000),
			inputs:          inputs,
			expectedOutputs: inputs,
		},
		// TODO_INCOMPLETE(#81): publisher channels which are full are proving harder to test
		// robustly (no flakiness); perhaps it has to do with the lack of some
		// kind of guarantee about the receiver order on the consumer side.
		//
		// The following scenarios should generally pass but are flaky:
		// (see: docs/pkg/observable/README.md regarding synchronization and buffering)
		//
		// {
		// 	name:            "full non-buffered publisher",
		// 	publishCh:       fullBlockingPublisher,
		// 	inputs:          inputs[1:],
		// 	expectedOutputs: inputs,
		// 	setupFn: func(t test) {
		// 		go func() {
		// 			// blocking send
		// 			t.publishCh <- &inputs[0]
		// 		}()
		// 	},
		// },
		// {
		// 	name:            "full buffered len 1 publisher",
		// 	publishCh:       fullBufferedPublisher,
		// 	inputs:          inputs[1:],
		// 	expectedOutputs: inputs,
		// 	setupFn: func(t test) {
		// 		// non-blocking send
		// 		t.publishCh <- &inputs[0]
		// 	},
		// },
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.setupFn != nil {
				test.setupFn(test)
			}

			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)

			obsvbl, publishCh := channel.NewObservable[int](
				channel.WithPublisher(test.publishCh),
			)
			require.NotNil(t, obsvbl)
			require.NotNil(t, publishCh)

			// construct 3 distinct observers, each with its own channel
			observers := make([]observable.Observer[int], 1)
			for i := range observers {
				observers[i] = obsvbl.Subscribe(ctx)
			}

			group, ctx := errgroup.WithContext(ctx)

			// ensure all observer channels are notified
			for obsvrIdx, obsvr := range observers {
				// onNext is called for each notification received by the observer
				onNext := func(outputIndex int, output int) error {
					// obsvr channel should receive notified input
					if !assert.Equalf(
						t, test.expectedOutputs[outputIndex],
						output,
						"obsvr Idx: %d", obsvrIdx,
					) {
						return testerrors.ErrAsync
					}
					return nil
				}

				// onDone is called when the observer channel closes
				onDone := func(outputs []int) error {
					if !assert.ElementsMatch(
						t, test.expectedOutputs, outputs,
						"obsvr addr: %p", obsvr,
					) {
						return testerrors.ErrAsync
					}
					return nil
				}

				// concurrently await notification or timeout to avoid blocking on
				// empty and/or non-buffered publishers.
				group.Go(goNotifiedOrTimedOutFactory(obsvr, onNext, onDone, notifyTimeout))
			}

			// notify with test input
			publish := delayedPublishFactory(publishCh, publishDelay)
			for _, input := range test.inputs {
				// simulating IO delay in sequential message publishing
				publish(input)
			}

			// Finished sending values, close publishCh to unsubscribe all observers
			// and close all fan-out channels.
			close(publishCh)

			// wait for obsvbl to be notified or timeout
			err := group.Wait()
			require.NoError(t, err)

			// closing publishCh should unsubscribe all observers, causing them
			// to close their channels.
			for _, observer := range observers {
				// must drain the channel first to ensure it is isClosed
				err := testchannel.DrainChannel(observer.Ch())
				require.NoError(t, err)
			}
		})
	}
}

func TestChannelObservable_UnsubscribeObservers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	obsvbl, publishCh := channel.NewObservable[int]()
	require.NotNil(t, obsvbl)
	require.NotNil(t, publishCh)

	type test struct {
		name        string
		lifecycleFn func() observable.Observer[int]
	}

	tests := []test{
		{
			name: "nil context",
			lifecycleFn: func() observable.Observer[int] {
				observer := obsvbl.Subscribe(nil)
				observer.Unsubscribe()
				return observer
			},
		},
		{
			name: "only unsubscribe",
			lifecycleFn: func() observable.Observer[int] {
				observer := obsvbl.Subscribe(ctx)
				observer.Unsubscribe()
				return observer
			},
		},
		{
			name: "only cancel",
			lifecycleFn: func() observable.Observer[int] {
				observer := obsvbl.Subscribe(ctx)
				cancel()
				return observer
			},
		},
		{
			// NOTE: this will log a warning that can be ignored:
			// >  redundant unsubscribe: observer is closed
			name: "cancel then unsubscribe",
			lifecycleFn: func() observable.Observer[int] {
				observer := obsvbl.Subscribe(ctx)
				cancel()
				time.Sleep(cancelUnsubscribeDelay)
				observer.Unsubscribe()
				return observer
			},
		},
		{
			// NOTE: this will log a warning that can be ignored:
			// >  redundant unsubscribe: observer is closed
			name: "unsubscribe then cancel",
			lifecycleFn: func() observable.Observer[int] {
				observer := obsvbl.Subscribe(ctx)
				observer.Unsubscribe()
				time.Sleep(cancelUnsubscribeDelay)
				cancel()
				return observer
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			observer := test.lifecycleFn()

			select {
			case value, ok := <-observer.Ch():
				require.Empty(t, value)
				require.False(t, ok)
			case <-time.After(notifyTimeout):
				t.Fatal("observer channel left open")
			}
		})
	}
}

// TODO_INCOMPLETE/TODO_TECHDEBT: Implement `TestChannelObservable_ConcurrentSubUnSub`
func TestChannelObservable_ConcurrentSubUnSub(t *testing.T) {
	t.Skip("add coverage: subscribing and unsubscribing concurrently should not race")
}

func TestChannelObservable_SequentialPublishAndUnsubscription(t *testing.T) {
	observations := new([]*observation[int])
	expectedNotifications := [][]int{
		{123, 456, 789},
		{456, 789, 987},
		{789, 987, 654},
		{987, 654, 321},
	}

	obsvbl, publishCh := channel.NewObservable[int]()
	require.NotNil(t, obsvbl)
	require.NotNil(t, publishCh)
	// simulate IO delay in sequential message publishing
	publish := delayedPublishFactory(publishCh, publishDelay)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	observation0 := newObservation(ctx, obsvbl)
	*observations = append(*observations, observation0)
	go goReceiveNotifications(observation0)
	publish(123)

	observation1 := newObservation(ctx, obsvbl)
	*observations = append(*observations, observation1)
	go goReceiveNotifications(observation1)
	publish(456)

	observation2 := newObservation(ctx, obsvbl)
	*observations = append(*observations, observation2)
	go goReceiveNotifications(observation2)
	publish(789)

	observation3 := newObservation(ctx, obsvbl)
	*observations = append(*observations, observation3)
	go goReceiveNotifications(observation3)

	observation0.Unsubscribe()
	publish(987)

	observation1.Unsubscribe()
	publish(654)

	observation2.Unsubscribe()
	publish(321)

	observation3.Unsubscribe()

	for obsnIdx, obsrvn := range *observations {
		t.Run(fmt.Sprintf("observation%d", obsnIdx), func(t *testing.T) {
			msg := "observer %d channel left open"
			select {
			case _, ok := <-obsrvn.Ch():
				require.Falsef(t, ok, msg, obsnIdx)
			default:
				t.Fatalf(msg, obsnIdx)
			}

			obsrvn.Lock()
			defer obsrvn.Unlock()

			require.EqualValuesf(
				t, expectedNotifications[obsnIdx], obsrvn.Notifications,
				"observation index: %d", obsnIdx,
			)
		})
	}
}

// TODO_TECHDEBT/TODO_INCOMPLETE: add coverage for active observers closing when publishCh closes.
func TestChannelObservable_ObserversCloseOnPublishChannelClose(t *testing.T) {
	t.Skip("add coverage: all observers should unsubscribe when publishCh closes")
}

type pubChWithMu struct {
	publishCh chan<- int
	mu        sync.Mutex
}

type obsWithMu struct {
	observable.Observable[int]
	mu sync.Mutex
}

// Defining TDD tests this way avoids race conditions
func TestChannelObservable_MergeObservables(t *testing.T) {
	runTestMergeObservables(
		t,
		"successful: merge 5 observables", // desc
		false,                             // failFast
		[]int{52, 12, 34, 423, 32},        // inputs
		[]int{12, 32, 34, 52, 423},        // expectedOutput
	)
	runTestMergeObservables(
		t,
		"successful: fast fail merge 5 observables, after 3 closes", // desc
		true,                       // failFast
		[]int{52, 12, 34, 423, 32}, // inputs
		[]int{12, 52},              // expectedOutput
	)
}

func runTestMergeObservables(t *testing.T, desc string, failFast bool, inputs []int, expectedOutput []int) {
	const testTimeOut = 2 * time.Second
	numObservables := len(inputs)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	observers := make([]observable.Observable[int], 0, numObservables)
	publishChs := make([]chan<- int, 0, numObservables)
	for i := 0; i < numObservables; i++ {
		observer, publishCh := channel.NewObservable[int]()
		observers = append(observers, observer)
		publishChs = append(publishChs, publishCh)
	}

	// Publish each input to each of the observables individually.
	// We do this in a goroutine to also start listening at the
	// same time on the merged observable.
	go func() {
		for i := 0; i < numObservables; i++ {
			if failFast {
				// Close the merged observable after 3 publishes
				// To simulate an obvserver closing/failing
				observers[len(expectedOutput)-1].UnsubscribeAll()
			}
			publishChs[i] <- inputs[i]
			t.Logf("published %d", inputs[i])
			time.Sleep(1 * time.Millisecond) // Allow publish to be processed
		}
	}()

	receivedAllCh := make(chan *struct{})
	receivedNotifications := make([]int, 0, len(inputs))
	recvNotifMutex := sync.RWMutex{}

	// Subscribe to the merged observable
	mergedObs := channel.Merge[int](ctx, observers, failFast)
	t.Cleanup(func() {
		mergedObs.UnsubscribeAll()
		cancel()
	})

	// Listen for notifications from the merged observables in a
	// goroutine so we can timeout the test if necessary.
	go func() {
		// Subscribe to the merged observable
		mergedObsCh := mergedObs.Subscribe(ctx)
		// Listen for notifications from the merged observable
		for notification := range mergedObsCh.Ch() {
			t.Logf("received %d", notification)
			recvNotifMutex.Lock()
			// Add the notification to the list of received notifications
			receivedNotifications = append(receivedNotifications, notification)
			// Stop listening if all notifications have been received
			if len(receivedNotifications) == len(expectedOutput) {
				recvNotifMutex.Unlock()
				receivedAllCh <- &struct{}{}
				return
			}
			recvNotifMutex.Unlock()
		}
	}()

	select {
	case <-receivedAllCh:
		sort.Ints(receivedNotifications)
		recvNotifMutex.RLock()
		defer recvNotifMutex.RUnlock()
		require.Equal(t, len(expectedOutput), len(receivedNotifications))
		require.EqualValuesf(
			t,
			expectedOutput,
			receivedNotifications,
			"expected %v, received %v", inputs, receivedNotifications,
		)
	case <-ctx.Done():
		t.Fatalf("test failed: context cancelled early")
	case <-time.After(testTimeOut):
		t.Fatalf("test failed: timed out after %v", testTimeOut)
	}
}

func delayedPublishFactory[V any](publishCh chan<- V, delay time.Duration) func(value V) {
	return func(value V) {
		publishCh <- value
		// simulate IO delay in sequential message publishing
		// NB: this make the test code safer as concurrent operations have more
		// time to react; i.e. interact with the test harness.
		time.Sleep(delay)
	}
}

func goNotifiedOrTimedOutFactory[V any](
	obsvr observable.Observer[V],
	onNext func(index int, output V) error,
	onDone func(outputs []V) error,
	timeoutDuration time.Duration,
) func() error {
	var (
		outputIndex int
		outputs     []V
	)
	return func() error {
		for {
			select {
			case output, ok := <-obsvr.Ch():
				if !ok {
					return onDone(outputs)
				}

				if err := onNext(outputIndex, output); err != nil {
					return err
				}

				outputs = append(outputs, output)
				outputIndex++
				continue
			case <-time.After(timeoutDuration):
				return fmt.Errorf("timed out waiting for observer to be notified")
			}
		}
	}
}
