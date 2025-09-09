package cmd

import (
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "gompc",
	Short: "Go MPC Utilities and Doctor check",
}

// Called by main.go
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	// Defaults
	viper.SetDefault("mpd.host", "127.0.0.1")
	viper.SetDefault("mpd.port", 6600)
	viper.SetDefault("mpd.timeout_ms", 2000)

	// Flags
	rootCmd.PersistentFlags().String("host", "", "MPD host (env MPD_HOST)")
	rootCmd.PersistentFlags().Int("port", 0, "MPD port (env MPD_PORT)")
	rootCmd.PersistentFlags().Int("timeout", 0, "Timeout ms (env YOURAPP_TIMEOUT_MS)")
	rootCmd.PersistentFlags().String("config", defaultConfigPath(), "Path to config file")

	_ = viper.BindPFlag("mpd.host", rootCmd.PersistentFlags().Lookup("host"))
	_ = viper.BindPFlag("mpd.port", rootCmd.PersistentFlags().Lookup("port"))
	_ = viper.BindPFlag("mpd.timeout_ms", rootCmd.PersistentFlags().Lookup("timeout"))
	_ = viper.BindPFlag("config_path", rootCmd.PersistentFlags().Lookup("config"))

	// env
	_ = viper.BindEnv("mpd.host", "MPD_HOST")
	_ = viper.BindEnv("mpd.port", "MPD_PORT")
	_ = viper.BindEnv("mpd.timeout_ms", "YOURAPP_TIMEOUT_MS")

	// Loads TOML config if present
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		path := viper.GetString("config_path")
		viper.SetConfigFile(path)
		viper.SetConfigType("toml")
		_ = viper.ReadInConfig()
	}
}

func defaultConfigPath() string {
	home, _ := os.UserHomeDir()
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "gompc", "config.toml")
	}
	return filepath.Join(home, ".config", "gompc", "config.toml")
}

func ms(d time.Duration) int64 {
	return d.Milliseconds()
}
