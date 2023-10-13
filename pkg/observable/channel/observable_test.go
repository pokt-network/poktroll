package channel_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"pocket/pkg/observable"
	"pocket/pkg/observable/channel"
)

const (
	notifyTimeout            = 100 * time.Millisecond
	unsubscribeSleepDuration = notifyTimeout * 2
)

func TestNewObservable_NotifyObservers(t *testing.T) {
	type test struct {
		name            string
		producer        chan *int
		inputs          []int
		expectedOutputs []int
		setupFn         func(t test)
	}

	inputs := []int{123, 456, 789}
	queuedProducer := make(chan *int, 1)
	nonEmptyBufferedProducer := make(chan *int, 1)

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
			name:            "queued non-buffered producer",
			producer:        queuedProducer,
			inputs:          inputs[1:],
			expectedOutputs: inputs,
			setupFn: func(t test) {
				go func() {
					// blocking send
					t.producer <- &inputs[0]
				}()
			},
		},
		{
			name:            "empty buffered len 1 producer",
			producer:        make(chan *int, 1),
			inputs:          inputs,
			expectedOutputs: inputs,
		},
		{
			name:            "non-empty buffered producer",
			producer:        nonEmptyBufferedProducer,
			inputs:          inputs[1:],
			expectedOutputs: inputs,
			setupFn: func(t test) {
				// non-blocking send
				t.producer <- &inputs[0]
			},
		},
	}

	for _, tt := range tests[:] {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn(tt)
			}

			// TECHDEBT/INCOMPLETE: test w/ & w/o context cancellation
			//ctx := context.Background()
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)

			t.Logf("producer: %p", tt.producer)
			testObs, producer := channel.NewObservable[*int](
				channel.WithProducer(tt.producer),
			)
			require.NotNil(t, testObs)
			require.NotNil(t, producer)

			// construct 3 distinct observers, each with its own channel
			observers := make([]observable.Observer[*int], 3)
			for i := range observers {
				observers[i] = testObs.Subscribe(ctx)
			}

			group, ctx := errgroup.WithContext(ctx)
			notifiedOrTimedOut := func(sub observable.Observer[*int]) func() error {
				var outputIndex int
				return func() error {
					for {
						select {
						case output, ok := <-sub.Ch():
							if !ok {
								return nil
							}

							// observer channel should receive notified input
							t.Logf("output: %d | %p", *output, output)
							require.Equal(t, tt.expectedOutputs[outputIndex], *output)
							outputIndex++
						case <-time.After(notifyTimeout):
							return fmt.Errorf("timed out waiting for observer to be notified")
						}
					}
				}
			}

			// ensure all observer channels are notified
			for _, observer := range observers {
				// concurrently await notification or timeout to avoid blocking on
				// empty and/or non-buffered producers.
				group.Go(notifiedOrTimedOut(observer))
			}

			// notify with test input
			t.Logf("sending to producer %p", producer)
			for i, input := range tt.inputs[:] {
				inputPtr := new(int)
				*inputPtr = input
				t.Logf("sending input ptr: %d %p", input, inputPtr)
				producer <- inputPtr
				t.Logf("send input %d", i)
			}
			cancel()

			// wait for testObs to be notified or timeout
			err := group.Wait()
			require.NoError(t, err)
			t.Log("errgroup done")

			// unsubscribing should close observer channel(s)
			for i, observer := range observers {
				observer.Unsubscribe()
				t.Logf("unsusbscribed %d", i)

				// must drain the channel first to ensure it is closed
				closed, err := drainCh(observer.Ch())
				require.NoError(t, err)
				require.True(t, closed)
			}
		})
	}
}

// TECHDEBT/INCOMPLETE: add coverage for multiple observers, unsubscribe from one
// and ensure the rest are still notified.

// TECHDEBT\INCOMPLETE: add coverage for active observers closing when producer closes.

func TestNewObservable_UnsubscribeObservers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	testObs, producer := channel.NewObservable[int]()
	require.NotNil(t, testObs)
	require.NotNil(t, producer)

	type test struct {
		name        string
		lifecycleFn func() observable.Observer[int]
	}

	tests := []test{
		{
			name: "nil context",
			lifecycleFn: func() observable.Observer[int] {
				observer := testObs.Subscribe(nil)
				observer.Unsubscribe()
				return observer
			},
		},
		{
			name: "only unsubscribe",
			lifecycleFn: func() observable.Observer[int] {
				observer := testObs.Subscribe(ctx)
				observer.Unsubscribe()
				return observer
			},
		},
		{
			name: "only cancel",
			lifecycleFn: func() observable.Observer[int] {
				observer := testObs.Subscribe(ctx)
				cancel()
				return observer
			},
		},
		{
			name: "cancel then unsubscribe",
			lifecycleFn: func() observable.Observer[int] {
				observer := testObs.Subscribe(ctx)
				cancel()
				time.Sleep(unsubscribeSleepDuration)
				observer.Unsubscribe()
				return observer
			},
		},
		{
			name: "unsubscribe then cancel",
			lifecycleFn: func() observable.Observer[int] {
				observer := testObs.Subscribe(ctx)
				observer.Unsubscribe()
				time.Sleep(unsubscribeSleepDuration)
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

func drainCh[V any](ch <-chan V) (closed bool, err error) {
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return true, nil
				return
			}
			continue
		default:
			return false, fmt.Errorf("observer channel left open")
		}
	}
}
