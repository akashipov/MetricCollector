package main

import (
	"fmt"
	"github.com/akashipov/MetricCollector/internal"
	"net/http"
)

func main() {
	if err := run(); err != nil {
		panic("Error is: " + err.Error())
	}
}

func run() error {
	internal.ParseArgsServer()
	fmt.Printf("Server is running on %s...\n", *internal.HPServer)
	return http.ListenAndServe(*internal.HPServer, internal.ServerRouter())
}
