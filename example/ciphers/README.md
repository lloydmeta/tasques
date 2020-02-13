## Ciphers

This is an example server-worker app.

### Start up a local Elasticsearch on port 9200

`docker run -p 9200:9200 --env discovery.type=single-node elasticsearch:7.5.2`

### Run the Tasques server

In the project root dir, start a Tasques server (`go run ./app`)

### Run the Ciphers server

On `example/ciphers/server`, run `go run main.go`

#### Add some Messages to cipher

Go to [localhost:8081](http://localhost:8081) (default) and create messages, which will create jobs.

### Start the worker

In `example/ciphers/server`, run `go run main.go --worker-id worker1` replacing `worker1` with a unique worker id 
per worker process.

#### Workers in other languages

There are example workers in Java and Rust as well.