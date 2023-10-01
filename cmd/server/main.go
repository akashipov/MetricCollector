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
	h := http.NewServeMux()
	h.Handle("/update/", http.HandlerFunc(internal.Update))
	return http.ListenAndServe(`:8080`, h)
}
