// Package polylog provides a ubiquitous logging interface which is derived from
// github.com/rs/zerolog, and as a result, also highly compatibly with other
// common, industry-standard logging libraries. This API mirrors that of zerolog
// but exists as a distinct layer of abstraction, and an extremely thin wrapper
// around the underlying logging library; especially in the case of the zerolog
// implementation. This distinction is intended to allow for evolution of the needs
// of this packages consumers as well as any future ambitions to (and implications
// thereof) adding support for adapting to additional logging libraries.
//
// It is intended to initially support the go std `log`, `github.com/rs/zerolog` and `go.uber.org/zap` logging libraries:
//
// - https://pkg.go.dev/log@go1.21.4
//
// - https://github.com/rs/zerolog
//
// - https://github.com/uber-go/zap
package polylog
