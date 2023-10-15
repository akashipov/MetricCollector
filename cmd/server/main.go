package main

import (
	"fmt"
	"github.com/akashipov/MetricCollector/internal/server"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan bool, 1)
	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Printf("Signal: %v\n", sig)
		done <- true
	}()
	go run()
	fmt.Println("awaiting signal")
	<-done
	fmt.Println("exiting")
}

func run() {
	server.ParseArgsServer()
	fmt.Printf("Server is running on %s...\n", *server.HPServer)
	err := http.ListenAndServe(*server.HPServer, server.ServerRouter())
	if err != nil {
		panic(err.Error())
	}
}
