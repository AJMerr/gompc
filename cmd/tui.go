package cmd

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/AJMerr/gompc/internal/app"
	"github.com/AJMerr/gompc/internal/mpd"
)

func init() {
	tuiCmd := &cobra.Command{
		Use:   "tui",
		Short: "Run the TUI music player",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Resolve MPD config from viper (flags/env/file already bound in root)
			cfg := mpd.Config{
				Host:    viper.GetString("mpd.host"),
				Port:    viper.GetInt("mpd.port"),
				Timeout: time.Duration(viper.GetInt("mpd.timeout_ms")) * time.Millisecond,
			}
			deps := app.Deps{
				Client: mpd.NewClient(), // TODO: implement
				Cfg:    cfg,
			}
			m := app.New(deps)
			p := tea.NewProgram(m, tea.WithAltScreen())
			_, err := p.Run()
			return err
		},
	}
	rootCmd.AddCommand(tuiCmd)
}
