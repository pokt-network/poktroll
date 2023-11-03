package client

import (
	"pocket/pkg/either"
	"pocket/pkg/observable"
)

// EventsQueryClientOption is an interface-wide type which can be implemented to use or modify the
// query client during construction. This would likely be done in an
// implementation-specific way; e.g. using a type assertion to assign to an
// implementation struct field(s).
type EventsQueryClientOption func(EventsQueryClient)

// TxClientOption defines a function type that modifies the TxClient. This pattern
// allows for flexible and optional configurations to be applied to a TxClient instance.
// Such options can be used to customize properties, behaviors, or any other attributes
// of the TxClient without altering its core construction logic.
type TxClientOption func(TxClient)

// BlocksObservable is an observable which is notified with an either
// value which contains either an error or the event message bytes.
//
// TODO_HACK: This type would be more useful as an alias
// (i.e. BlocksObservable = observable.Observable[Block]); however, so long as
// gomock is a testing dependency, and it continues not to support generic types,
// this must be its own non-generic type.
type BlocksObservable observable.ReplayObservable[Block]

// EventsBytesObservable is an observable which is notified with an either
// value which contains either an error or the event message bytes.
//
// TODO_HACK: This type would be more useful as an alias
// (i.e. EventsBytesObservable = observable.Observable[either.Bytes]); however,
// so long as gomock is a testing dependency, and it continues not to support
// generic types, this must be its own non-generic type.
type EventsBytesObservable observable.Observable[either.Bytes]
