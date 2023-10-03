package main

import (
	"github.com/akashipov/MetricCollector/internal"
	"github.com/go-resty/resty/v2"
)

func main() {
	a := internal.MetricSender{
		URL:         "http://localhost:8080",
		ListMetrics: &internal.ListMetrics,
		Client:      resty.New(),
	}
	a.PollInterval(false)
}
