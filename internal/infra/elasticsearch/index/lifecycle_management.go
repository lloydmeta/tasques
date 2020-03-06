package index

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/rs/zerolog/log"

	"github.com/lloydmeta/tasques/internal/config"
	"github.com/lloydmeta/tasques/internal/infra/elasticsearch/common"
	"github.com/lloydmeta/tasques/internal/infra/elasticsearch/task"
)

var (
	DefaultArchivedTasquesPolicyName = "tasques_archived_tasks_policy"
	DefaulttArchivedTasquesPolicy    = Json{
		"phases": Json{
			"hot": Json{
				"actions": Json{
					"rollover": Json{
						"max_size": "50GB",
						"max_age":  "1d",
					},
				},
			},
			"warm": Json{
				"min_age": "7d",
				"actions": Json{
					"readonly": Json{},
				},
			},
			"cold": Json{
				"min_age": "30d",
				"actions": Json{
					"freeze": Json{},
				},
			},
		},
	}
)

type ILMSetup interface {

	// A Check to make sure that, if enabled, the setup was previously carried out
	// Does NOT make sure the existing setup, if present, is up to date !
	Check(ctx context.Context) error

	// Returns a function for modifying the Index Template for Archived Tasks
	ArchivedTemplateHook() func(t *Template)

	// Performs ILM setup, if required
	InstallPolicies(ctx context.Context) error

	// Bootstraps the indices that were set up for ILM
	BootstrapIndices(ctx context.Context) error
}

type ilmSetupImpl struct {
	esClient *elasticsearch.Client
	config   config.LifecycleSetup
}

func NewILMSetup(esClient *elasticsearch.Client, config config.LifecycleSetup) ILMSetup {
	return &ilmSetupImpl{
		esClient: esClient,
		config:   config,
	}
}

func (i *ilmSetupImpl) ArchivedTemplateHook() func(t *Template) {
	return func(t *Template) {
		if i.config.ArchivedTasks.Enabled {
			policyName := i.archivedTasksPolicyName()
			t.Patterns = []Pattern{fmt.Sprintf("%s-*", task.TasquesArchiveIndex)}
			if t.Settings == nil {
				t.Settings = make(Json)
			}
			t.Settings["index.lifecycle.name"] = policyName
			t.Settings["index.lifecycle.rollover_alias"] = task.TasquesArchiveIndex
		}
	}
}

func (i *ilmSetupImpl) InstallPolicies(ctx context.Context) error {
	if i.config.ArchivedTasks.Enabled {
		policyName := i.archivedTasksPolicyName()
		asBytes, err := json.Marshal(i.archivedTasksPolicy())
		if err != nil {
			return common.JsonSerdesErr{Underlying: []error{err}}
		}
		log.Info().RawJSON("body", asBytes).Str("policy_name", string(policyName)).Msg("Applying lifecycle policy")
		req := esapi.ILMPutLifecycleRequest{
			Policy: policyName,
			Body:   bytes.NewReader(asBytes),
		}
		rawResp, err := req.Do(ctx, i.esClient)
		if err != nil {
			return common.ElasticsearchErr{Underlying: err}
		}
		defer rawResp.Body.Close()

		switch rawResp.StatusCode {
		case 200:
			return nil
		default:
			return common.UnexpectedEsStatusError(rawResp)
		}

	} else {
		return nil
	}
}

func (i *ilmSetupImpl) Check(ctx context.Context) error {
	if i.config.ArchivedTasks.Enabled {
		policyName := i.archivedTasksPolicyName()
		req := esapi.ILMGetLifecycleRequest{
			Policy: policyName,
		}
		rawResp, err := req.Do(ctx, i.esClient)
		if err != nil {
			return common.ElasticsearchErr{Underlying: err}
		}
		defer rawResp.Body.Close()

		switch rawResp.StatusCode {
		case 200:
			return nil
		case 404:
			return PolicyNotInstalled{NotInstalled: []string{policyName}}
		default:
			return common.UnexpectedEsStatusError(rawResp)
		}

	} else {
		return nil
	}
}

func (i *ilmSetupImpl) archivedTasksPolicyName() string {
	if i.config.ArchivedTasks.CustomPolicy != nil {
		return i.config.ArchivedTasks.CustomPolicy.Name
	} else {
		return DefaultArchivedTasquesPolicyName
	}
}
func (i *ilmSetupImpl) archivedTasksPolicy() Json {
	if i.config.ArchivedTasks.CustomPolicy != nil {
		return Json{
			"policy": i.config.ArchivedTasks.CustomPolicy.Policy,
		}
	} else {
		return Json{
			"policy": DefaulttArchivedTasquesPolicy,
		}
	}
}

func (i *ilmSetupImpl) BootstrapIndices(ctx context.Context) error {
	if i.config.ArchivedTasks.Enabled {
		indexCreateBody := buildWriteAlias(task.TasquesArchiveIndex)
		indexCreateBodyAsBytes, err := json.Marshal(indexCreateBody)
		if err != nil {
			return common.JsonSerdesErr{Underlying: []error{err}}
		}
		req := esapi.IndicesCreateRequest{
			Index: fmt.Sprintf("%%3C%s-%%7Bnow%%2Fd%%7D-000001%%3E", task.TasquesArchiveIndex),
			Body:  bytes.NewReader(indexCreateBodyAsBytes),
		}
		rawResp, err := req.Do(ctx, i.esClient)
		if err != nil {
			return common.ElasticsearchErr{Underlying: err}
		}
		defer rawResp.Body.Close()
		respStatus := rawResp.StatusCode
		switch {
		case 200 <= respStatus || respStatus <= 299:
			return nil
		default:
			return common.UnexpectedEsStatusError(rawResp)
		}
	} else {
		return nil
	}
}

type PolicyNotInstalled struct {
	NotInstalled []string
}

func (t PolicyNotInstalled) Error() string {
	return fmt.Sprintf("One or more app ILM policies were not installed. Please run the setup command to install them [%v]", t.NotInstalled)
}

func buildWriteAlias(alias string) Json {
	return Json{
		"aliases": Json{
			alias: Json{
				"is_write_index": true,
			},
		},
	}
}
