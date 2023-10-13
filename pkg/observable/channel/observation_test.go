package channel_test

import (
	"context"
	"sync"

	"pocket/pkg/observable"
)

type observation[V any] struct {
	sync.Mutex
	observable.Observer[V]
	Notifications *[]V
}

func newObservation[V any](
	ctx context.Context,
	observable observable.Observable[V],
) *observation[V] {
	return &observation[V]{
		Observer:      observable.Subscribe(ctx),
		Notifications: new([]V),
	}
}

func (o *observation[V]) notify(value V) {
	o.Lock()
	defer o.Unlock()

	*o.Notifications = append(*o.Notifications, value)
}

func goReceiveNotifications[V any](obsvn *observation[V]) {
	for notification := range obsvn.Ch() {
		obsvn.notify(notification)
	}
}
