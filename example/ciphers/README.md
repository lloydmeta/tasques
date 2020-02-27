## Ciphers

This is an example server-worker app.

### Setup

If you have k8s installed, you can use the Makefile in the `k8s` dir to install ECK, spin up the needed infra,
and get credentials that you can put into the relevant config files.

If you don't you'll need to spin these up separately.

### Run the Tasques server

In the project root dir, start a Tasques server (`go run ./app`)

### Run the Ciphers server

On `example/ciphers/server`, run `go run main.go`

#### Add some Messages to cipher

Go to the server at [localhost:9000](http://localhost:9000) (default) and create messages, which will create jobs.

### Start the Go worker

In `example/ciphers/worker-go`, run `go run main.go --worker-id worker1` replacing `worker1` with a unique worker id 
per worker process.

#### Workers in other languages

There are example workers in Java and Rust as well (`worker-$lang`)