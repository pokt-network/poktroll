// Package config provides a method by which dependencies can be injected into
// dependency chains, via the use of SupplierFn functions. These functions
// return functions that can be used in CLI code to chain the dependencies
// required to start a service into a single depinject.Config.
package config
