package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/akashipov/MetricCollector/internal/agent"
	"github.com/go-resty/resty/v2"
)

func run(wg *sync.WaitGroup, done chan bool) {
	agent.ParseArgsClient()
	client := resty.New()
	client = client.SetTimeout(2 * time.Second)
	var m sync.Mutex
	a := agent.MetricSender{
		URL:                fmt.Sprintf("http://%s", *agent.HPClient),
		ListMetrics:        &agent.ListMetrics,
		Client:             client,
		ReportIntervalTime: agent.ReportInterval,
		PollIntervalTime:   agent.PollInterval,
		M:                  &m,
		Done:               done,
	}
	memInfo := runtime.MemStats{}
	var countOfUpdate atomic.Int64
	wg.Add(1)
	go func() {
		fmt.Println("Has been started PollInterval")
		a.PollInterval(&memInfo, &countOfUpdate, done)
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		fmt.Println("Has been started ReportInterval")
		a.ReportInterval(false, &memInfo, &countOfUpdate, done)
		wg.Done()
	}()
	wg.Done()
}

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan bool)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		close(done)
		wg.Done()
	}()
	wg.Add(1)
	go run(&wg, done)
	fmt.Println("Awaiting signal...")
	_, isRunning := <-done
	if isRunning {
		fmt.Println("Something is wrong")
	}
	fmt.Println("Exiting...")
	wg.Wait()
}
