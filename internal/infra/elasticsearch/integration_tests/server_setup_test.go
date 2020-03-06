// +build integration

package integration_tests

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lloydmeta/tasques/internal/config"
	"github.com/lloydmeta/tasques/internal/infra/server"
)

func Test_Server_Setup(t *testing.T) {

	ArchivedTasksLock.Lock()
	defer ArchivedTasksLock.Unlock()

	if err := deleteArchivedTasksIndex(); err != nil {
		t.Error(err)
		return
	}

	client := NewTestClient(func(req *http.Request) *http.Response {
		assert.EqualValues(t, req.URL.String(), "http://localhost_dummy/api/saved_objects/_import?overwrite=true")
		return &http.Response{
			StatusCode: 200,
			// Send response to be tested
			Body: ioutil.NopCloser(bytes.NewBufferString(`{"success": true}`)),
			// Must be set to non-nil value or it panics
			Header: make(http.Header),
		}
	})
	setup := server.NewSetup(client, esClient, &dummyAppConfig)

	err := setup.Check(ctx)
	assert.Error(t, err)

	err = setup.RunIfNeeded(ctx)
	assert.NoError(t, err)

	err = setup.Check(ctx)
	assert.NoError(t, err)

}

var dummyAppConfig = config.App{
	KibanaClient: &config.KibanaClient{
		Address: "http://localhost_dummy",
	},
	LifecycleSetup: config.LifecycleSetup{
		ArchivedTasks: config.LifecycleSettings{
			Enabled: true,
		},
	},
}

// RoundTripFunc .
type RoundTripFunc func(req *http.Request) *http.Response

// RoundTrip .
func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

//NewTestClient returns *http.Client with Transport replaced to avoid making real calls
func NewTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: fn,
	}
}
