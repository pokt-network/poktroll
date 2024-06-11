package common

import (
	"context"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

type MultiKeeper interface {
	ApplicationKeeper
	SupplierKeeper
}

type ApplicationKeeper interface {
	SetApplication(context.Context, apptypes.Application)
}

type SupplierKeeper interface {
	SetSupplier(context.Context, sharedtypes.Supplier)
}

type SessionKeeper interface {
	GetSession(context.Context, *sessiontypes.QueryGetSessionRequest) (*sessiontypes.QueryGetSessionResponse, error)
	StoreBlockHash(ctx context.Context)
}
