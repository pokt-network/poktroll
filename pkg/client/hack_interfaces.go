//go:generate mockgen -destination=../../testutil/mockclient/block_client_mock.go -package=mockclient . BlockClient
//go:generate mockgen -destination=../../testutil/mockclient/delegation_client_mock.go -package=mockclient . DelegationClient
package client

import (
	"context"

	"github.com/pokt-network/poktroll/pkg/observable"
)

// TODO_HACK: The purpose of these type is to work around gomock's lack of
// support for generic types. Both of these clients are implemented as
// MappedClient[T] objects, which being a generic cannot be mocked with
// gomock. For the same reason, these cannot be aliases
// (i.e. type BlockClient = MappedClient[Block]).
// They cannot also be direct definitions of the implemented type
// (i.e. type BlockClient MappedClient[Block])
// This is a limitation of gomock, and other mocking tools should be considered

type BlockObservable observable.ReplayObservable[Block]

type BlockClient interface {
	EventsSequence(context.Context) BlockObservable
	LatestEvent(context.Context) Block
	Close()
}

type DelegateeChangeObservable observable.ReplayObservable[DelegateeChange]

type DelegationClient interface {
	EventsSequence(context.Context) DelegateeChangeObservable
	LatestEvent(context.Context) DelegateeChange
	Close()
}
