// Package rings provides the RingCache interface that is used to build rings
// for applications by either the application itself or a gateway. This ring
// is used to sign the relay requests.
// The RingCache is responsible for caching the rings for future use and
// invalidating the cache when the application's delegated gateways change.
package rings
