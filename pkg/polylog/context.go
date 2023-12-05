package polylog

// CtxKey is the key used to store the polylog.Logger in a context.Context. This
// is **independant** of any logger-implementation-specific context key that may
// be used internal to any of the logger implementations. Polylog attempts to
// provide a ubiquitous interface for storing and retrieving loggers from the
// context but also to integrate with the underlying logger implementations as
// seamlessly as possible.
const CtxKey = "polylog/context"
