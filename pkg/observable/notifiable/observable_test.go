package notifiable_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"pocket/pkg/observable"
	"pocket/pkg/observable/notifiable"
)

const (
	notifyTimeout            = 100 * time.Millisecond
	unsubscribeSleepDuration = notifyTimeout * 2
)

func TestNewNotifiableObservable(t *testing.T) {
	type test struct {
		name     string
		notifier chan int
	}

	input := 123
	nonEmptyBufferedNotifier := make(chan int, 1)
	nonEmptyBufferedNotifier <- input

	tests := []test{
		{name: "nil notifier", notifier: nil},
		{name: "empty non-buffered notifier", notifier: make(chan int)},
		{name: "empty buffered len 1 notifier", notifier: make(chan int, 1)},
		{name: "non-empty buffered notifier", notifier: nonEmptyBufferedNotifier},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			notifee, notifier := notifiable.NewObservable[int](tt.notifier)
			require.NotNil(t, notifee)
			require.NotNil(t, notifier)

			// construct 3 distinct subscriptions, each with its own channel
			subscriptions := make([]observable.Subscription[int], 3)
			for i := range subscriptions {
				subscriptions[i] = notifee.Subscribe(ctx)
			}

			group := errgroup.Group{}
			notifiedOrTimedOut := func(subscriptionCh <-chan int) func() error {
				return func() error {
					// subscriptionCh should receive notified input
					select {
					case output := <-subscriptionCh:
						require.Equal(t, input, output)
					case <-time.After(notifyTimeout):
						return fmt.Errorf("timed out waiting for subscription to be notified")
					}
					return nil
				}
			}

			// ensure all subscription channels are notified
			for _, subscription := range subscriptions {
				// concurrently await notification or timeout to avoid blocking on
				// empty and/or non-buffered notifiers.
				group.Go(notifiedOrTimedOut(subscription.Ch()))
			}

			// notify with test input
			notifier <- input

			// wait for notifee to be notified or timeout
			err := group.Wait()
			require.NoError(t, err)

			// unsubscribing should close subscription channel(s)
			for _, subscription := range subscriptions {
				subscription.Unsubscribe()

				select {
				case <-subscription.Ch():
				default:
					t.Fatal("subscription channel left open")
				}
			}
		})
	}
}

func TestSubscription_Unsubscribe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	notifee, notifier := notifiable.NewObservable[int](nil)
	require.NotNil(t, notifee)
	require.NotNil(t, notifier)

	tests := []struct {
		name        string
		lifecycleFn func() observable.Subscription[int]
	}{
		{
			name: "nil context",
			lifecycleFn: func() observable.Subscription[int] {
				subscription := notifee.Subscribe(nil)
				subscription.Unsubscribe()
				return subscription
			},
		},
		{
			name: "only unsubscribe",
			lifecycleFn: func() observable.Subscription[int] {
				subscription := notifee.Subscribe(ctx)
				subscription.Unsubscribe()
				return subscription
			},
		},
		{
			name: "only cancel",
			lifecycleFn: func() observable.Subscription[int] {
				subscription := notifee.Subscribe(ctx)
				cancel()
				return subscription
			},
		},
		{
			name: "cancel then unsubscribe",
			lifecycleFn: func() observable.Subscription[int] {
				subscription := notifee.Subscribe(ctx)
				cancel()
				time.Sleep(unsubscribeSleepDuration)
				subscription.Unsubscribe()
				return subscription
			},
		},
		{
			name: "unsubscribe then cancel",
			lifecycleFn: func() observable.Subscription[int] {
				subscription := notifee.Subscribe(ctx)
				subscription.Unsubscribe()
				time.Sleep(unsubscribeSleepDuration)
				cancel()
				return subscription
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subscription := tt.lifecycleFn()

			select {
			case value, ok := <-subscription.Ch():
				require.Empty(t, value)
				require.False(t, ok)
			case <-time.After(notifyTimeout):
				t.Fatal("subscription channel left open")
			}
		})
	}
}
