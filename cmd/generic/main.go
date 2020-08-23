package main

import (
	"github.com/bakito/batch-job-controller/cmd"
	"github.com/bakito/batch-job-controller/pkg/http"
)

func main() {
	main := cmd.Setup()
	main.Start(
		http.StaticFileServer(8080, main.Config.ReportDirectory),
		http.GenericAPIServer(main.Config.CallbackServicePort, main.Config.ReportDirectory),
	)
}
