package main

import (
	"context"
	"log"

	"github.com/bakito/batch-job-controller/cmd"
	"github.com/bakito/batch-job-controller/pkg/http"
)

func main() {
	cmd.SetupLogger(false)

	log.Fatal(http.MockAPIServer(8090).
		Start(context.TODO()))
}
