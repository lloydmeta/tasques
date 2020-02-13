package cmd

import (
	"context"
	"net/http"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/lloydmeta/tasques/internal/config"
	"github.com/lloydmeta/tasques/internal/infra/elasticsearch/common"
	"github.com/lloydmeta/tasques/internal/infra/elasticsearch/index"
	"github.com/lloydmeta/tasques/internal/infra/kibana/saved_objects"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Run tasques setup",
	Long:  "Runs various setup routines for Tasques. Includes Index Templates, and Kibana saved objects (if configured)",
	Run: func(cmd *cobra.Command, args []string) {
		log.Info().Msg("Setting up Index templates")
		templatesSetup, err := buildTemplatesSetup(appConfig.Elasticsearch)
		if err != nil {
			log.Fatal().Err(err).Msg("Could not setup Template Setter Upper")
		}
		if err := templatesSetup.Run(context.Background()); err != nil {
			log.Fatal().Err(err).Msg("Failed to install index templates")
		}
		if appConfig.KibanaClient != nil {
			log.Info().Msg("Setting up Kibana saved objects")
			httpClient := http.Client{}
			importer := saved_objects.NewImporter(*appConfig.KibanaClient, &httpClient)
			if err := importer.Run(context.Background()); err != nil {
				log.Fatal().Err(err).Msg("Failed to install Kibana saved objects")
			}
		}
		log.Info().Msg("Setup complete.")
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

func buildTemplatesSetup(conf config.ElasticsearchClient) (*index.TemplatesSetup, error) {
	esClient, err := common.NewClient(conf)
	if err != nil {
		return nil, err
	} else {
		setup := index.DefaultTemplateSetup(esClient)
		return &setup, nil
	}
}
