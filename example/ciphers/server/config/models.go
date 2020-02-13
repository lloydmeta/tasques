package config

import (
	config2 "github.com/lloydmeta/tasques/example/ciphers/common/config"
	"github.com/lloydmeta/tasques/worker/config"
)

type App struct {
	BindAddress   string                      `json:"bind_address" mapstructure:"bind_address"`
	Elasticsearch config2.ElasticsearchClient `json:"elasticsearch" mapstructure:"elasticsearch"`
	CipherQueuer  CipherQueuer                `json:"cipher_queuer" mapstructure:"cipher_queuer"`
}

type CipherQueuer struct {
	Server config.TasquesServer `json:"server" mapstructure:"server"`
	Queue  string               `json:"queue" mapstructure:"queue"`
}
