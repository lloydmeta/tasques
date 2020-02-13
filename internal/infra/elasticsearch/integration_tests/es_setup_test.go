// +build integration

// This package holds a single TestMain method that does setup and teardown
// of a shared ES container for running integration tests against
package integration_tests

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/ory/dockertest"
)

// esClient holds an elasticearch.Client that is filled in when TestMain is invoked,
// after the docker container has been set up
var esClient *elasticsearch.Client

func TestMain(m *testing.M) {
	// uses a sensible default on windows (tcp/http) and linux/osx (socket)
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	// pulls an image, creates a container based on it and runs it
	options := dockertest.RunOptions{
		Repository: "elasticsearch",
		Tag:        "7.5.2",
		Env:        []string{"discovery.type=single-node"},
	}
	resource, err := pool.RunWithOptions(&options)
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}
	hostPort := resource.GetPort("9200/tcp")

	connectionString := fmt.Sprintf("http://localhost:%s", hostPort)

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	if err := pool.Retry(func() error {
		var err error
		esClient, err = elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{connectionString}})
		if err != nil {
			return err
		}
		healthRequest := esapi.ClusterHealthRequest{}
		if resp, err := healthRequest.Do(context.Background(), esClient); err != nil {
			return err
		} else {
			if resp.IsError() {
				return fmt.Errorf("response was an error [%v]", resp)
			}
		}
		return nil
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	code := m.Run()

	// You can't defer this because os.Exit doesn't care for defer
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	os.Exit(code)
}
