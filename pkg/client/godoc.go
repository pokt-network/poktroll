// Package client defines interfaces and types that facilitate interactions
// with blockchain functionalities, both transactional and observational. It is
// built to provide an abstraction layer for sending, receiving, and querying
// blockchain data, thereby offering a standardized way of integrating with
// various blockchain platforms.
//
// The client package leverages external libraries like cosmos-sdk and cometbft,
// but there is a preference to minimize direct dependencies on these external
// libraries, when defining interfaces, aiming for a cleaner decoupling.
// It seeks to provide a flexible and comprehensive interface layer, adaptable to
// different blockchain configurations and requirements.
package client
