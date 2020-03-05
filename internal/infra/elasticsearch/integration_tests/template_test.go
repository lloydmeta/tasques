// +build integration

package integration_tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lloydmeta/tasques/internal/config"
	"github.com/lloydmeta/tasques/internal/infra/elasticsearch/index"
)

func Test_DefaultTemplatesSetup_Run(t *testing.T) {
	ilmSetup := index.NewILMSetup(esClient, lifecycleSetupSettings)
	subject := index.DefaultTemplateSetup(esClient, ilmSetup.ArchivedTemplateHook())

	err := subject.Run(context.Background())
	assert.NoError(t, err)

	err = subject.Check(context.Background())
	assert.NoError(t, err)

}

var lifecycleSetupSettings = config.LifeCycleSetup{
	ArchivedTasks: config.LifeCycleSettings{
		Enabled: true,
	},
}
