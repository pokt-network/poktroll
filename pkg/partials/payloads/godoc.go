// Package payloads contains the different types of RPC payloads the partials
// package supports. The structs defined here are used to partially unmarshal
// the payload and extract the minimal fields required to: generate error
// responses, retrieve RPC request type, determine request compute units, etc...
// This is done through partially unmarshalling the payload into the minimum
// required set of pre-defined fields that need to be explicitly determined for
// each RPC type supported.
package payloads
