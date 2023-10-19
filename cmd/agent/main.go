package main

import (
	"fmt"
	"github.com/akashipov/MetricCollector/internal/agent"
	"github.com/go-resty/resty/v2"
	"os"
	"os/signal"
	"syscall"
)

func run() {
	agent.ParseArgsClient()
	a := agent.MetricSender{
		URL:                fmt.Sprintf("http://%s", *agent.HPClient),
		ListMetrics:        &agent.ListMetrics,
		Client:             resty.New(),
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
