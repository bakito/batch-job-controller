package main

import (
	"context"
	"flag"
	"log"

	"github.com/bakito/batch-job-controller/cmd"
	"github.com/bakito/batch-job-controller/pkg/http"
)

func main() {
	port := flag.Int("port", 8090, "define the port the mock runs on")
	json := flag.Bool("json-logs", false, "enable to log in json format (default: false)")
	isoTime := flag.Bool("iso-time", true, "enable to log time in ISO format (default: true) if false, epoch format is used")
	flag.Parse()

	cmd.SetupLogger(*json, *isoTime)

	log.Fatal(http.MockAPIServer(*port).Start(context.TODO()))
}
