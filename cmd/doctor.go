package cmd

import (
	"context"
	"os"
	"time"

	"github.com/AJMerr/gompc/internal/doctor"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	var deep, jsonOut bool

	doctorCmd := &cobra.Command{
		Use:   "doctor",
		Short: "Runs connectivity and readiness checks.",
		RunE: func(cmd *cobra.Command, args []string) error {
			host := viper.GetString("mpd.host")
			port := viper.GetInt("mpd.port")
			timeoutMS := viper.GetInt("mpd.timeout_ms")

			cfg := doctor.Config{
				Host:      host,
				Port:      port,
				TimeoutMS: timeoutMS,
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMS)*time.Millisecond)
			defer cancel()

			rep := doctor.Run(ctx, cfg, deep)
			if jsonOut {
				doctor.RenderJSON(rep)
			} else {
				doctor.RenderHuman(cfg, viper.ConfigFileUsed(), rep)
			}
			if rep.ExitCode != 0 {
				os.Exit(rep.ExitCode)
			}
			return nil
		},
	}

	doctorCmd.Flags().BoolVar(&deep, "deep", false, "Run deeper (idle/noidle) checks")
	doctorCmd.Flags().BoolVar(&jsonOut, "json", false, "Output JSON")

	rootCmd.AddCommand(doctorCmd)
}
