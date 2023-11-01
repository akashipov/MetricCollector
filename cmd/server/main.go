package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/akashipov/MetricCollector/internal/server"
	"go.uber.org/zap"
)

func main() {
	var sugar zap.SugaredLogger
	logger, err := zap.NewDevelopment()
	if err != nil {
		// вызываем панику, если ошибка
		panic(err)
	}
	defer logger.Sync()
	sugar = *logger.Sugar()
	srv := &http.Server{Handler: server.ServerRouter(&sugar)}

	done := make(chan bool, 1)

	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigint
		fmt.Println()
		fmt.Printf("Signal: %v\n", sig)
		done <- true

		// We received an interrupt signal, shut down.
		if err := srv.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout:
			log.Printf("HTTP server Shutdown: %v", err)
		}
		close(idleConnsClosed)
	}()
	go run(srv)

	fmt.Println("awaiting signal")
	<-done
	fmt.Println("Has got signal")
	<-idleConnsClosed
	fmt.Println("exiting")
}

func Storage() {
	tickerStorageInterval := time.NewTicker(time.Duration(*server.PTSave) * time.Second)
	defer tickerStorageInterval.Stop()
	for {
		select {
		case <-tickerStorageInterval.C:
			f, err := os.Create(*server.FSPath)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			b, err := json.Marshal(*server.MapMetric)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			f.Write(b)
			fmt.Println("Metrics are saved!")
		}
	}
}

func run(srv *http.Server) {
	server.ParseArgsServer()
	srv.Addr = *server.HPServer
	fmt.Printf("Server is running on %s...\n", *server.HPServer)
	go Storage()
	if *server.StartLoadMetric {
		b, err := os.ReadFile(*server.FSPath)
		if err == nil {
			err = json.Unmarshal(b, server.MapMetric)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			fmt.Println("Metrics are successfully loaded..")
		}
	}
	err := srv.ListenAndServe()
	if err != http.ErrServerClosed {
		log.Fatalf("HTTP server ListenAndServe: %v", err)
	}
}
