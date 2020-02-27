package index

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/rs/zerolog/log"

	"github.com/lloydmeta/tasques/internal/infra/elasticsearch/common"
	"github.com/lloydmeta/tasques/internal/infra/elasticsearch/leader"
	"github.com/lloydmeta/tasques/internal/infra/elasticsearch/task"
)

type TemplateName string
type Pattern = string
type Json = map[string]interface{}
type Mappings = map[string]interface{}

// Template defines a template to be applied when setup is run
type Template struct {
	name     TemplateName // ignored when serialising because the name doesn't start with a capital
	Patterns []Pattern    `json:"index_patterns"`
	Mappings Mappings     `json:"mappings,omitempty"`
}

func (t *Template) Name() TemplateName {
	return t.name
}

func NewTemplate(name TemplateName, patterns []Pattern, mappings Mappings) Template {
	return Template{name: name, Patterns: patterns, Mappings: mappings}
}

// TemplatesSetup holds a list of Templates and has the ability to actually
// send them to the server
type TemplatesSetup struct {
	esClient  *elasticsearch.Client
	Templates []Template
}

// Returns the default Template setter upper
func DefaultTemplateSetup(esClient *elasticsearch.Client) TemplatesSetup {
	return TemplatesSetup{
		esClient: esClient,
		Templates: []Template{
			TasquesQueuesTemplate,
			TasquesLocksTemplate,
			TasquesRecurringTasksTemplate,
		},
	}
}

// Runs the setup
func (s *TemplatesSetup) Run(ctx context.Context) error {
	var errors []error
	for _, template := range s.Templates {
		if err := s.putTemplate(ctx, &template); err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) != 0 {
		return PutTemplateErrors{Errors: errors}
	} else {
		return nil
	}
}

// Checks if the current TemplatesSetup was run.
//
// This is currently a shallow check for template presence only.
func (s *TemplatesSetup) Check(ctx context.Context) error {
	indexTemplateNames := make([]string, 0, len(s.Templates))
	for _, t := range s.Templates {
		indexTemplateNames = append(indexTemplateNames, string(t.Name()))
	}

	indexTemplatesGetReq := esapi.IndicesGetTemplateRequest{Name: indexTemplateNames}

	rawResp, err := indexTemplatesGetReq.Do(context.Background(), s.esClient)
	if err != nil {
		return common.ElasticsearchErr{Underlying: err}
	}
	defer rawResp.Body.Close()
	switch rawResp.StatusCode {
	case 200:
		var mappings map[string]interface{}
		if err = json.NewDecoder(rawResp.Body).Decode(&mappings); err != nil {
			return common.JsonSerdesErr{Underlying: []error{err}}
		}
		var notPresent []string
		for _, name := range indexTemplateNames {
			if _, ok := mappings[name]; !ok {
				notPresent = append(notPresent, name)
			}
		}
		if len(notPresent) != 0 {
			return TemplatesNotInstalled{NotInstalled: notPresent}
		} else {
			return nil
		}
	case 404:
		return TemplatesNotInstalled{NotInstalled: indexTemplateNames}
	default:
		return common.UnexpectedEsStatusError(rawResp)
	}
}

func (s *TemplatesSetup) putTemplate(ctx context.Context, t *Template) error {
	asBytes, err := json.Marshal(t)
	log.Info().RawJSON("body", asBytes).Str("template_name", string(t.name)).Msg("Applying template")
	if err != nil {
		return common.JsonSerdesErr{Underlying: []error{err}}
	}
	putTemplateReq := esapi.IndicesPutTemplateRequest{
		Body: bytes.NewReader(asBytes),
		Name: string(t.name),
	}
	rawResp, err := putTemplateReq.Do(ctx, s.esClient)
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
}

type PutTemplateErrors struct {
	Errors []error
}

func (e PutTemplateErrors) Error() string {
	return fmt.Sprintf("Errors encountered [%v]", e.Errors)
}

type TemplatesNotInstalled struct {
	NotInstalled []string
}

func (t TemplatesNotInstalled) Error() string {
	return fmt.Sprintf("One or more app index templates were not installed. Please run the setup command to install them [%v]", t.NotInstalled)
}

// Templates

// Tasques queues template (disables dynamic for most object fields to
// prevent mapping conflicts and mapping explosions)
var TasquesQueuesTemplate = NewTemplate(
	".tasques_queues_index_template",
	[]Pattern{Pattern(task.BuildIndexName("*"))},
	Mappings{
		"_source": Json{
			"enabled": true,
		},
		"dynamic": true, // We use persistence models anyways, so we can make sure mappings don't  get out of hand
		"properties": Json{
			"retry_times": Json{
				"type": "long",
			},
			"remaining_attempts": Json{
				"type": "long",
			},
			"kind": Json{
				"type": "text",
				"fields": Json{
					"keyword": Json{
						"type":         "keyword",
						"ignore_above": 256,
					},
				},
			},
			"state": Json{
				"type": "keyword",
			},
			"run_at": Json{
				"type": "date",
			},
			"processing_timeout": Json{
				"type": "long",
			},
			"priority": Json{
				"type": "long",
			},
			"args": Json{
				"type":    "object",
				"enabled": false, // free-form JSON from users don't get indexed to prevent explosions
			},
			"context": Json{
				"type":    "object",
				"enabled": false,
			},
			"last_enqueued_at": Json{
				"type": "date",
			},
			"last_claimed": Json{
				"properties": Json{
					"worker_id": Json{
						// Allows use to search by thisl
						"type": "text",
						"fields": Json{
							"keyword": Json{
								"type":         "keyword",
								"ignore_above": 256,
							},
						},
					},
					"claimed_at": Json{
						"type": "date",
					},
					"times_out_at": Json{
						"type": "date",
					},
					"last_report": Json{
						"properties": Json{
							"at": Json{
								"type": "date",
							},
							"data": Json{
								"type":    "object",
								"enabled": false,
							},
						},
					},
					"result": Json{
						"properties": Json{
							"at": Json{
								"type": "date",
							},
							"failure": Json{
								"type":    "object",
								"enabled": false,
							},
							"success": Json{
								"type":    "object",
								"enabled": false,
							},
						},
					},
				},
			},
			"metadata": Json{
				"properties": Json{
					"created_at": Json{
						"type": "date",
					},
					"modified_at": Json{
						"type": "date",
					},
				},
			},
		},
	},
)

// Just sets source to enabled and dynamic to true since we own this
var TasquesLocksTemplate = NewTemplate(
	".tasques_leader_locks_index_template",
	[]Pattern{Pattern(leader.IndexName)},
	Mappings{
		"_source": Json{
			"enabled": true,
		},
		"dynamic": true,
	},
)

// Tasques recurring tasks template (disables dynamic for user-defined object fields to
// prevent mapping conflicts and mapping explosions)
var TasquesRecurringTasksTemplate = NewTemplate(
	".tasques_recurring_tasks_index_template",
	[]Pattern{".tasques_recurring_tasks"},
	Mappings{
		"_source": Json{
			"enabled": true,
		},
		"dynamic": true, // We use persistence models anyways, so we can make sure mappings don't  get out of hand
		"properties": Json{
			"is_deleted": Json{
				"type": "boolean",
			},
			"loaded_at": Json{
				"type": "date",
			},
			"metadata": Json{
				"properties": Json{
					"created_at": Json{
						"type": "date",
					},
					"modified_at": Json{
						"type": "date",
					},
				},
			},
			"schedule_expression": Json{
				"type": "text",
				"fields": Json{
					"keyword": Json{
						"type":         "keyword",
						"ignore_above": 256,
					},
				},
			},
			"task_definition": Json{
				"properties": Json{
					"kind": Json{
						"type": "text",
						"fields": Json{
							"keyword": Json{
								"type":         "keyword",
								"ignore_above": 256,
							},
						},
					},
					"priority": Json{
						"type": "long",
					},
					"args": Json{
						"type":    "object",
						"enabled": false, // free-form JSON from users don't get indexed to prevent explosions
					},
					"context": Json{
						"type":    "object",
						"enabled": false,
					},
					"processing_timeout": Json{
						"type": "long",
					},
					"queue": Json{
						"type": "text",
						"fields": Json{
							"keyword": Json{
								"type":         "keyword",
								"ignore_above": 256,
							},
						},
					},
					"retry_times": Json{
						"type": "long",
					},
				},
			},
		},
	},
)
