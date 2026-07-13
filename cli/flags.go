package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type FlagData struct {
	Prompt         bool
	IgnoreExisting bool
	RadarrUrl      string
	RadarrApiKey   string
	RadarrBasePath string
	RadarrPathMaps []string
}

func configureFlags(root *cobra.Command) error {
	flags := FlagData{}
	pflags := root.PersistentFlags()

	pflags.BoolVarP(&flags.Prompt, "prompt", "p", false, "prompt for confirmation before each file operation")
	pflags.BoolVarP(&flags.IgnoreExisting, "ignore-existing", "i", false, "skip items that already exist at the destination")
	pflags.StringVar(&flags.RadarrUrl, "radarr-url", "", "Radarr API URL (e.g. http://localhost:7878)")
	pflags.StringVar(&flags.RadarrApiKey, "radarr-api-key", "", "Radarr API Key")
	pflags.StringVar(&flags.RadarrBasePath, "radarr-base-path", "", "Base path for Radarr (e.g. /mnt/video)")
	pflags.StringArrayVar(&flags.RadarrPathMaps, "radarr-path-map", nil, "Map Radarr path segments to local (e.g. documentary=docu), repeatable")

	// binding map for viper/pflag -> env
	m := map[string]string{
		"prompt":           "INGEST_PROMPT",
		"ignore-existing":  "INGEST_IGNORE_EXISTING",
		"radarr-url":       "RADARR_URL",
		"radarr-api-key":   "RADARR_API_KEY",
		"radarr-base-path": "RADARR_BASE_PATH",
		"radarr-path-map":  "",
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
		Prompt:         viper.GetBool("prompt"),
		IgnoreExisting: viper.GetBool("ignore-existing"),
		RadarrUrl:      viper.GetString("radarr-url"),
		RadarrApiKey:   viper.GetString("radarr-api-key"),
		RadarrBasePath: viper.GetString("radarr-base-path"),
		RadarrPathMaps: viper.GetStringSlice("radarr-path-map"),
	}
}
