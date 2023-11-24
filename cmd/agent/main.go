package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/akashipov/MetricCollector/internal/agent"
	"github.com/go-resty/resty/v2"
)

func run() {
	agent.ParseArgsClient()
	client := resty.New()
	client = client.SetTimeout(2 * time.Second)
	a := agent.MetricSender{
		URL:                fmt.Sprintf("http://%s", *agent.HPClient),
		ListMetrics:        &agent.ListMetrics,
		Client:             client,
		ReportIntervalTime: agent.ReportInterval,
		PollIntervalTime:   agent.PollInterval,
	}
	a.PollInterval(false)
}

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan bool, 1)
	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		done <- true
	}()
	go run()
	fmt.Println("awaiting signal")
	<-done
	fmt.Println("exiting")
}
