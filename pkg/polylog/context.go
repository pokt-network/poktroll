package polylog

import "context"

// CtxKey is a type which is intended to disambiguate polylog context keys from
// other context keys which may be used in the same context.Context.
type CtxKey string

// PolylogCtxKey is the key used to store the polylog.Logger in a context.Context.
// This is **independent** of any logger-implementation-specific context key that
// may be used internal to any of the logger implementations. Polylog attempts to
// provide a ubiquitous interface for storing and retrieving loggers from the
// context but also to integrate with the underlying logger implementations as
// seamlessly as possible.
const PolylogCtxKey CtxKey = "polylog/context"

// DefaultContextLogger is the default logger implementation used when no logger
// is associated with a context. It is assigned in the implementation package's
// init() function to avoid potentially creating import cycles.
// The default logger implementation is zerolog (i.e. pkg/polylog/polyzero).
//
// IMPORTANT: In order for the default to be populated, the polyzero package MUST
// be part of the build. Otherwise, the polyzero package's init function will
// neither be included in the build nor executed. If no such import exists, the
// polyzero package can be imported for side effects only, e.g.:
//
//	import _ "github.com/pokt-network/pocket/pkg/polylog/polyzero"
var DefaultContextLogger Logger

// Ctx returns the Logger associated with the ctx. If no logger is associated,
// DefaultContextLogger is returned, unless DefaultContextLogger is nil, in which
// case a disabled logger is returned.
//
// To get a context which is associated a given logger, call the respective logger's
// #WithContext() method. Then this function can be used to retrieve it from that
// (or a context derived from that) context, later and elsewhere.
func Ctx(ctx context.Context) Logger {
	logger, ok := ctx.Value(PolylogCtxKey).(Logger)
	if !ok {
		return DefaultContextLogger
	}
	return logger
}
