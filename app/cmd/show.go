package cmd

import (
	"encoding/json"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(showCmd)
	showCmd.AddCommand(showConfigCmd)
}

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show information",
	Long:  `Sometimes you just need to know more`,
}

var showConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Show config",
	Long:  `Renders the config that we end up using`,
	Run: func(cmd *cobra.Command, args []string) {
		out, err := json.MarshalIndent(&appConfig, "", "  ")
		if err != nil {
			log.Fatal().Err(err).Msg("Error marshalling config to JSON")
		} else {
			log.Info().Msg(string(out))
		}
	},
}
