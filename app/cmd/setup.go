package cmd

import (
	"context"
	"net/http"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/lloydmeta/tasques/internal/infra/elasticsearch/common"
	"github.com/lloydmeta/tasques/internal/infra/elasticsearch/index"
	"github.com/lloydmeta/tasques/internal/infra/kibana/saved_objects"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Run tasques setup",
	Long:  "Runs various setup routines for Tasques. Includes Index Templates, and Kibana saved objects (if configured)",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		esClient, err := common.NewClient(appConfig.Elasticsearch)
		if err != nil {
			log.Fatal().Err(err).Msg("Could not setup Elasticsearch client")
		}
		log.Info().Msg("Setting up ILM")
		ilmSetup := index.NewILMSetup(esClient, appConfig.LifecycleSetup)
		if err := ilmSetup.InstallPolicies(ctx); err != nil {
			log.Fatal().Err(err).Msg("Could not install ILM policies")
		}

		// This needs to happen after ILM because templates refer to ILM policies
		log.Info().Msg("Setting up Index templates")
		templatesSetup := index.DefaultTemplateSetup(esClient, ilmSetup.ArchivedTemplateHook())
		if err != nil {
			log.Fatal().Err(err).Msg("Could not setup Template Setter Upper")
		}
		if err := templatesSetup.Run(ctx); err != nil {
			log.Fatal().Err(err).Msg("Failed to install index templates")
		}

		if err := ilmSetup.BootstrapIndices(ctx); err != nil {
			log.Fatal().Err(err).Msg("Could not bootstrap indices for ILM")
		}

		if appConfig.KibanaClient != nil {
			log.Info().Msg("Setting up Kibana saved objects")
			httpClient := http.Client{}
			importer := saved_objects.NewImporter(*appConfig.KibanaClient, &httpClient)
			if err := importer.Run(ctx); err != nil {
				log.Fatal().Err(err).Msg("Failed to install Kibana saved objects")
			}
		}
		log.Info().Msg("Setup complete.")
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
}
