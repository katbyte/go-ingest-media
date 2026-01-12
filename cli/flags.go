package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type FlagData struct {
	Confirm        bool
	IgnoreExisting bool
}

func configureFlags(root *cobra.Command) error {
	flags := FlagData{}
	pflags := root.PersistentFlags()

	pflags.BoolVarP(&flags.Confirm, "confirm", "c", false, "")
	pflags.BoolVarP(&flags.IgnoreExisting, "ignore-existing", "i", false, "")

	// binding map for viper/pflag -> env
	m := map[string]string{
		"confirm":         "INGEST_CONFIRM",
		"ignore-existing": "INGEST_IGNORE_EXISTING",
	}

	for name, env := range m {
		if err := viper.BindPFlag(name, pflags.Lookup(name)); err != nil {
			return fmt.Errorf("error binding '%s' flag: %w", name, err)
		}

		if env != "" {
			if err := viper.BindEnv(name, env); err != nil {
				return fmt.Errorf("error binding '%s' to env '%s' : %w", name, env, err)
			}
		}
	}

	return nil
}

func GetFlags() FlagData {
	return FlagData{
		Confirm:        viper.GetBool("confirm"),
		IgnoreExisting: viper.GetBool("ignore-existing"),
	}
}
