package config

import (
	config2 "github.com/lloydmeta/tasques/example/ciphers/common/config"
	"github.com/lloydmeta/tasques/worker/config"
)

type App struct {
	CipherWorker  CipherWorker                `json:"cipher_worker" mapstructure:"cipher_worker"`
	Elasticsearch config2.ElasticsearchClient `json:"elasticsearch" mapstructure:"elasticsearch"`
}

type CipherWorker struct {
	Queue         string               `json:"queue" mapstructure:"queue"`
	TasquesWorker config.TasquesWorker `json:"tasques_worker" mapstructure:"tasques_worker"`
}
