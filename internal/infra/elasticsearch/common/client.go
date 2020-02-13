package common

import (
	"net/http"

	"github.com/elastic/go-elasticsearch/v8"
	"go.elastic.co/apm/module/apmelasticsearch"

	"github.com/lloydmeta/tasques/internal/config"
)

// NewClient returns a configured elasticsearch.Client based on the given conf
func NewClient(conf config.ElasticsearchClient) (*elasticsearch.Client, error) {
	wrappedTransport := apmelasticsearch.WrapRoundTripper(http.DefaultTransport)
	esClientConfig := elasticsearch.Config{Addresses: conf.Addresses, Transport: wrappedTransport}
	if conf.User != nil {
		esClientConfig.Username = conf.User.Name
		esClientConfig.Password = conf.User.Password
	}

	esClient, err := elasticsearch.NewClient(esClientConfig)
	if err != nil {
		return nil, err
	} else {
		return esClient, nil
	}
}
