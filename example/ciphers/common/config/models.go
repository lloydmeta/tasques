package config

import (
	"github.com/elastic/go-elasticsearch/v8"

	"github.com/lloydmeta/tasques/worker/config"
)

type ElasticsearchClient struct {
	Addresses []string              `json:"addresses" mapstructure:"addresses"`
	User      *config.BasicAuthUser `json:"user,omitempty" mapstructure:"user"`
}

func (e *ElasticsearchClient) BuildClient() (*elasticsearch.Client, error) {
	esClientConfig := elasticsearch.Config{Addresses: e.Addresses}
	if e.User != nil {
		esClientConfig.Username = e.User.Name
		esClientConfig.Password = e.User.Password
	}
	return elasticsearch.NewClient(esClientConfig)
}
