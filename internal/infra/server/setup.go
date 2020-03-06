package server

import (
	"context"
	"net/http"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/rs/zerolog/log"

	"github.com/lloydmeta/tasques/internal/config"
	"github.com/lloydmeta/tasques/internal/infra/elasticsearch/index"
	"github.com/lloydmeta/tasques/internal/infra/kibana/saved_objects"
)

// Setup abstracts away:
//
// 1. Setting up the environment for running Tasques
// 2. Checking that things are set up
type Setup interface {

	// Check returns an error if all the necessary setup is not complete
	Check(ctx context.Context) error

	// RunIfNeeded attempts to run the subroutines necessary, no more no less
	RunIfNeeded(ctx context.Context) error
}

type impl struct {
	ilm            index.ILMSetup
	templateSetup  index.TemplateSetup
	kibanaImporter saved_objects.Importer
}

// NewSetup returns a Setup implementation
func NewSetup(httpClient *http.Client, esClient *elasticsearch.Client, config *config.App) Setup {

	ilmSetup := index.NewILMSetup(esClient, config.LifecycleSetup)
	templateSetup := index.DefaultTemplateSetup(esClient, ilmSetup.ArchivedTemplateHook())
	kibanaImporter := saved_objects.NewImporter(*config.KibanaClient, httpClient)

	return &impl{
		ilm:            ilmSetup,
		templateSetup:  templateSetup,
		kibanaImporter: kibanaImporter,
	}
}

func (i *impl) Check(ctx context.Context) error {
	if err := i.ilm.Check(ctx); err != nil {
		return err
	} else if err := i.templateSetup.Check(ctx); err != nil {
		return err
	} else {
		return nil
	}
}

func (i *impl) RunIfNeeded(ctx context.Context) error {

	needsIlmSetup := false
	if err := i.ilm.Check(ctx); err != nil {
		if _, policiesNotFound := err.(index.PolicyNotInstalled); policiesNotFound {
			needsIlmSetup = true
		} else {
			log.Info().Msg("Skipping ILM setup")
			return err
		}
	}

	if needsIlmSetup {
		log.Info().Msg("Setting up ILM")
		if err := i.ilm.InstallPolicies(ctx); err != nil {
			log.Error().Err(err).Msg("Could not install ILM policies")
			return err
		}
	}

	if err := i.templateSetup.Check(ctx); err != nil {
		if _, templateNotFound := err.(index.TemplatesNotInstalled); templateNotFound {
			log.Info().Msg("Setting up Index templates")
			if err := i.templateSetup.Run(ctx); err != nil {
				log.Error().Err(err).Msg("Failed to install index templates")
				return err
			}
			log.Info().Msg("Installing Kibana Saved Objects")
			if err := i.kibanaImporter.Run(ctx); err != nil {
				log.Error().Err(err).Msg("Failed to install Kibana saved objects")
				return err
			}
		}
	}

	if needsIlmSetup {
		log.Info().Msg("Bootstrapping ILM indices")
		if err := i.ilm.BootstrapIndices(ctx); err != nil {
			log.Error().Err(err).Msg("Could not bootstrap indices for ILM")
			return err
		}
	}

	log.Info().Msg("Setup complete")
	return nil
}
