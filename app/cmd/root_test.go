package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_initConfig(t *testing.T) {
	if wd, err := os.Getwd(); err != nil {
		t.Error(err)
	} else {
		configFile = wd + "/../../config/tasques.example.yaml"
	}
	initConfig()
	assert.EqualValues(t, "passw0rd", appConfig.Elasticsearch.User.Password)
}
