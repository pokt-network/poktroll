package channel_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"pocket/internal/testchannel"
	"pocket/pkg/observable"
	"pocket/pkg/observable/channel"
)

const (
	productionDelay        = 2 * time.Millisecond
	notifyTimeout          = 50 * time.Millisecond
	cancelUnsubscribeDelay = notifyTimeout * 2
)

func TestChannelObservable_NotifyObservers(t *testing.T) {
	type test struct {
		name            string
		producer        chan *int
		inputs          []int
		expectedOutputs []int
		setupFn         func(t test)
	}

	inputs := []int{123, 456, 789}
	// NB: see INCOMPLETE comment below
	// fullBlockingProducer := make(chan *int)
	// fullBufferedProducer := make(chan *int, 1)

	tests := []test{
		{
			name:            "nil producer",
			producer:        nil,
			inputs:          inputs,
			expectedOutputs: inputs,
		},
		{
			name:            "empty non-buffered producer",
			producer:        make(chan *int),
			inputs:          inputs,
			expectedOutputs: inputs,
		},
		{
			name:            "empty buffered len 1 producer",
			producer:        make(chan *int, 1),
			inputs:          inputs,
			expectedOutputs: inputs,
		},
		// INCOMPLETE: producer channels which are full are proving harder to test
		// robustly (no flakiness); perhaps it has to do with the lack of some
		// kind of guarantee about the receiver order on the consumer side.
		//
		// The following scenarios should generally pass but are flaky:
		//
		// {
		// 	name:            "full non-buffered producer",
		// 	producer:        fullBlockingProducer,
		// 	inputs:          inputs[1:],
		// 	expectedOutputs: inputs,
		// 	setupFn: func(t test) {
		// 		go func() {
		// 			// blocking send
		// 			t.producer <- &inputs[0]
		// 		}()
		// 	},
		// },
		// {
		// 	name:            "full buffered len 1 producer",
		// 	producer:        fullBufferedProducer,
		// 	inputs:          inputs[1:],
		// 	expectedOutputs: inputs,
		// 	setupFn: func(t test) {
		// 		// non-blocking send
		// 		t.producer <- &inputs[0]
		// 	},
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn(tt)
			}

			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)

			obsvbl, producer := channel.NewObservable[*int](
				channel.WithProducer(tt.producer),
			)
			require.NotNil(t, obsvbl)
			require.NotNil(t, producer)
			produce := produceWithDelay(producer, productionDelay)

			// construct 3 distinct observers, each with its own channel
			observers := make([]observable.Observer[*int], 1)
			for i := range observers {
				observers[i] = obsvbl.Subscribe(ctx)
			}

			group, ctx := errgroup.WithContext(ctx)

			// ensure all obsvr channels are notified
			for obsvrIdx, obsvr := range observers {
				next := func(outputIndex int, output *int) error {
					// obsvr channel should receive notified input
					if !assert.Equalf(
						t, tt.expectedOutputs[outputIndex],
						*output,
						"obsvr Idx: %d", obsvrIdx,
					) {
						return fmt.Errorf("unexpected output")
					}
					return nil
				}

				done := func(outputs []*int) error {
					if !assert.Equalf(
						t, len(tt.expectedOutputs),
						len(outputs),
						"obsvr addr: %p", obsvr,
					) {
						return fmt.Errorf("unexpected number of outputs")
					}
					return nil
				}

				// concurrently await notification or timeout to avoid blocking on
				// empty and/or non-buffered producers.
				group.Go(goNotifiedOrTimedOutFactory(obsvr, next, done, notifyTimeout))
			}

			// notify with test input
			for _, input := range tt.inputs {
				inputPtr := new(int)
				*inputPtr = input

				// simulating IO delay in sequential message production
				produce(inputPtr)
			}
			cancel()

			// wait for obsvbl to be notified or timeout
			err := group.Wait()
			require.NoError(t, err)

			// unsubscribing should close obsvr channel(s)
			for _, observer := range observers {
				observer.Unsubscribe()

				// must drain the channel first to ensure it is closed
				err := testchannel.DrainChannel(observer.Ch())
				require.NoError(t, err)
			}
		})
	}
}

func TestChannelObservable_UnsubscribeObservers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	obsvbl, producer := channel.NewObservable[int]()
	require.NotNil(t, obsvbl)
	require.NotNil(t, producer)

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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			observer := tt.lifecycleFn()

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

func TestChannelObservable_ConcurrentSubUnSub(t *testing.T) {
	t.Skip("add coverage: subscribing and unsubscribing concurrently should not race")
}

func TestChannelObservable_SequentialProductionAndUnsubscription(t *testing.T) {
	observations := new([]*observation[int])
	expectedNotifications := [][]int{
		{123, 456, 789},
		{456, 789, 987},
		{789, 987, 654},
		{987, 654, 321},
	}

	obsvbl, producer := channel.NewObservable[int]()
	require.NotNil(t, obsvbl)
	require.NotNil(t, producer)
	// simulate IO delay in sequential message production
	produceWithDelay := produceWithDelay(producer, productionDelay)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	observation0 := newObservation(ctx, obsvbl)
	*observations = append(*observations, observation0)
	go goReceiveNotifications(observation0)
	produceWithDelay(123)

	observation1 := newObservation(ctx, obsvbl)
	*observations = append(*observations, observation1)
	go goReceiveNotifications(observation1)
	produceWithDelay(456)

	observation2 := newObservation(ctx, obsvbl)
	*observations = append(*observations, observation2)
	go goReceiveNotifications(observation2)
	produceWithDelay(789)

	observation3 := newObservation(ctx, obsvbl)
	*observations = append(*observations, observation3)
	go goReceiveNotifications(observation3)

	observation0.Unsubscribe()
	produceWithDelay(987)

	observation1.Unsubscribe()
	produceWithDelay(654)

	observation2.Unsubscribe()
	produceWithDelay(321)

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

			require.Equalf(
				t, len(expectedNotifications[obsnIdx]),
				len(*obsrvn.Notifications),
				"observation index: %d, expected: %+v, actual: %+v",
				obsnIdx, expectedNotifications[obsnIdx], *obsrvn.Notifications,
			)
			for notificationIdx, expected := range expectedNotifications[obsnIdx] {
				require.Equalf(
					t, expected,
					(*obsrvn.Notifications)[notificationIdx],
					"allExpected: %+v, allActual: %+v",
					expectedNotifications[obsnIdx], *obsrvn.Notifications,
				)
			}
		})
	}
}

// TECHDEBT/INCOMPLETE: add coverage for active observers closing when producer closes.
func TestChannelObservable_ObserversCloseOnProducerClose(t *testing.T) {
	t.Skip("add coverage: all observers should close when producer closes")
}

func produceWithDelay[V any](producer chan<- V, delay time.Duration) func(value V) {
	return func(value V) {
		time.Sleep(delay / 2)
		producer <- value
		time.Sleep(delay / 2)
	}
}

func goNotifiedOrTimedOutFactory[V any](
	obsvr observable.Observer[V],
	next func(index int, output V) error,
	done func(outputs []V) error,
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
					return done(outputs)
				}

				if err := next(outputIndex, output); err != nil {
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
