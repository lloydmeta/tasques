package cmd

import (
	"context"
	"net/http"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/lloydmeta/tasques/internal/infra/elasticsearch/common"
	"github.com/lloydmeta/tasques/internal/infra/server"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Run tasques setup",
	Long:  "Runs various setup routines for Tasques. Includes Index Templates, Kibana saved objects, and ILM setup (if configured)",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		httpClient := http.Client{}
		esClient, err := common.NewClient(appConfig.Elasticsearch)
		if err != nil {
			log.Fatal().Err(err).Msg("Could not setup Elasticsearch client")
		}

		setup := server.NewSetup(&httpClient, esClient, &appConfig)
		if err := setup.RunIfNeeded(ctx); err != nil {
			log.Fatal().Err(err).Msg("Setup failed")
		}
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
}
