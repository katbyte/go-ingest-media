package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type FlagData struct {
	BaseSrcPath string
	BaseDstPath string
}

func configureFlags(root *cobra.Command) error {
	flags := FlagData{}
	pflags := root.PersistentFlags()

	pflags.StringVarP(&flags.BaseSrcPath, "base-in", "i", "/mnt/ztmp/torrents/sorted", "")
	pflags.StringVarP(&flags.BaseDstPath, "base-out", "o", "/mnt/video", "")
	// pflags.BoolVarP(&flags.FullFetch, "full", "f", false, "do a full fetch and not abort")

	// binding map for viper/pflag -> env
	m := map[string]string{
		"base-in":  "BASE_IN_PATH",
		"base-out": "BASE_OUT_PATH",
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
	// there has to be an easier way....
	return FlagData{
		BaseSrcPath: viper.GetString("base-in"),
		BaseDstPath: viper.GetString("base-out"),
	}
}
