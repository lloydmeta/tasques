// +build integration

package integration_tests

import (
	"testing"

	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/stretchr/testify/assert"

	"github.com/lloydmeta/tasques/internal/infra/elasticsearch/index"
	"github.com/lloydmeta/tasques/internal/infra/elasticsearch/task"
)

func Test_IlmSetup(t *testing.T) {
	ArchivedTasksLock.Lock()
	defer ArchivedTasksLock.Unlock()

	if err := deleteArchivedTasksIndex(); err != nil {
		t.Error(err)
		return
	}

	setup := index.NewILMSetup(esClient, lifecycleSetupSettings)

	var NotInstalled index.PolicyNotInstalled
	err := setup.Check(ctx)
	assert.IsType(t, NotInstalled, err)

	assert.NotNil(t, setup.ArchivedTemplateHook())

	err = setup.InstallPolicies(ctx)
	assert.NoError(t, err)

	err = setup.BootstrapIndices(ctx)
	assert.NoError(t, err)

	err = setup.Check(ctx)
	assert.NoError(t, err)
}

func deleteArchivedTasksIndex() error {
	req := esapi.IndicesDeleteRequest{
		Index:             []string{task.TasquesArchiveIndex},
		AllowNoIndices:    esapi.BoolPtr(true),
		IgnoreUnavailable: esapi.BoolPtr(true),
	}
	rawResp, err := req.Do(ctx, esClient)
	defer rawResp.Body.Close()
	if err != nil {
		return err
	} else {
		return nil
	}

}
