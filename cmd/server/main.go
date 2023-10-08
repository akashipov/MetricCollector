package main

import (
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
	return http.ListenAndServe(*internal.HPServer, internal.ServerRouter())
}
