package config

import (
	"context"

	"cosmossdk.io/depinject"
	"github.com/spf13/cobra"
)

// SupplierFn is a function that is used to supply a depinject config.
type SupplierFn func(
	context.Context,
	depinject.Config,
	*cobra.Command,
) (depinject.Config, error)
