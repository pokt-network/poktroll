package flags

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type FlagDescriptor struct {
	FlagName    string
	ConfigKey   string
	Description string
}

func BindFlags(cmd *cobra.Command, flagDescriptors ...FlagDescriptor) error {
	v := viper.GetViper()

	for _, flagDesc := range flagDescriptors {
		// Register a persistent flag on the command.
		cmd.PersistentFlags().String(
			flagDesc.FlagName,
			v.GetString(flagDesc.ConfigKey),
			flagDesc.Description,
		)

		// Bind the flag to the respective config key in viper.
		if err := v.BindPFlag(
			flagDesc.ConfigKey,
			cmd.PersistentFlags().Lookup(flagDesc.FlagName),
		); err != nil {
			return err
		}
	}
	return nil
}
