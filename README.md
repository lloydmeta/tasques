## Tasques [![Build Status](https://travis-ci.org/lloydmeta/tasques.svg?branch=master)](https://travis-ci.org/lloydmeta/tasques) [![codecov](https://codecov.io/gh/lloydmeta/tasques/branch/master/graph/badge.svg)](https://codecov.io/gh/lloydmeta/tasques) [![](https://images.microbadger.com/badges/commit/lloydmeta/tasques.svg)](https://microbadger.com/images/lloydmeta/tasques "tasques docker image details")

Task queues backed by ES: Tasques.

### Features:

- Easily scalable:
  - Servers are stateless; easily spin more up as needed
  - The storage engine is Elasticsearch, nuff' said.
- Tasks can be configured
  - Priority
  - When to run
  - Retries
    - Tasks can be configured to retry X times, with exponential increase in run times
- Timeouts
  - Tasks that are picked up by workers that either don't report in or finish on time get timed out.
- Unclaiming
  - Tasks that were picked up but can't be handled now can be requeued without consequence.
- Recurring Tasks
  - Tasks that are repeatedly enqueued at configurable intervals (cron format with basic macro support)

### Requirements

1. Go 1.13+

### Usage

#### Running

1. Go to `docker/k8s` and run `make install-eck deploy`, and wait until the pods are all ready.
2. For Swagger, go to [localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)
    ![Swagger](swagger.png)

There is also an example project that demonstrates the application-tasques-worker relationship more thoroughly; please
see `example/ciphers` for more details.

##### APM

The server supports APM, as configured according to the [official docs](https://www.elastic.co/guide/en/apm/agent/go/current/getting-started.html#configure-setup).

#### Dev

1. [Install `Go`](https://golang.org/doc/install)
2. Use your favourite editor/IDE
3. For updating Swagger docs:
    1. Install [Swaggo](https://github.com/swaggo/swag#getting-started)
    2. Run `swag init -g app/main.go` from the root project dir
        * Check that there are no `time.Time` fields... there's a race condition in there somewhere
    3. Commit the generated files.
4. For updating the Go Client:
    1. Install [go-swagger](https://goswagger.io/generate/client.html)
    2. Run `swagger generate client -f docs/swagger.yaml`
    3. Commit the generated files.