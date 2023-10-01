package main

import (
	"github.com/akashipov/MetricCollector/internal"
)

func main() {
	a := internal.MetricSender{
		URL:         "http://localhost:8080",
		ListMetrics: &internal.ListMetrics,
	}
	a.PollInterval(false)
}
