package main

import (
	"fmt"
	"github.com/akashipov/MetricCollector/internal"
	"github.com/go-resty/resty/v2"
)

func main() {
	internal.ParseArgsClient()
	a := internal.MetricSender{
		URL:                fmt.Sprintf("http://%s", *internal.HPClient),
		ListMetrics:        &internal.ListMetrics,
		Client:             resty.New(),
		ReportIntervalTime: internal.ReportInterval,
		PollIntervalTime:   internal.PollInterval,
	}
	a.PollInterval(false)
}
