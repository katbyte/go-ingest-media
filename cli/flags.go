package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type FlagData struct {
	BaseSrcPath    string
	BaseDstPath    string
	Confirm        bool
	IgnoreExisting bool
}

func configureFlags(root *cobra.Command) error {
	flags := FlagData{}
	pflags := root.PersistentFlags()

	pflags.StringVarP(&flags.BaseSrcPath, "src", "s", "/mnt/ztmp/torrents/sorted", "")
	pflags.StringVarP(&flags.BaseDstPath, "dst", "d", "/mnt/video", "")
	pflags.BoolVarP(&flags.Confirm, "confirm", "c", false, "")
	pflags.BoolVarP(&flags.IgnoreExisting, "ignore-existing", "i", false, "")

	// binding map for viper/pflag -> env
	m := map[string]string{
		"src":             "INGEST_SRC_PATH",
		"dst":             "INGEST_DST_PATH",
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
	// there has to be an easier way....
	return FlagData{
		BaseSrcPath:    viper.GetString("src"),
		BaseDstPath:    viper.GetString("dst"),
		Confirm:        viper.GetBool("confirm"),
		IgnoreExisting: viper.GetBool("ignore-existing"),
	}
}
