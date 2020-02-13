package main

import "github.com/lloydmeta/tasques/app/cmd"

func main() {
	cmd.Execute()
}

// @title Tasques API
// @version 0.0.1
// @description A Task queue backed by Elasticsearch

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @host localhost:8080
// @securityDefinitions.basic BasicAuth
// @BasePath /
