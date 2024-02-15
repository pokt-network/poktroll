// Package polyzero provides a polylog.Logger implementation backed by zerolog.
// As the polylogger interface mirrors that of zerolog, this package is a thin
// wrapper around the zerolog package. However, it is only a partial mapping of
// the full zerolog API, and has already begun to deviate a bit to be more
// accommodating to other supported logging libraries.
//
// Use polyzero if you don't have a preference for a particular logging library
// or are already using zerolog.
package polyzero
