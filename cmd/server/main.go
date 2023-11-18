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

	"github.com/akashipov/MetricCollector/internal/general"
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
		<-tickerStorageInterval.C
		var f *os.File
		var err error
		fun := func() error {
			f, err = os.Create(*server.FSPath)
			return err
		}
		err = general.RetryCode(fun, syscall.EACCES)
		if err != nil {
			fmt.Println("Create block: " + err.Error())
			return
		}
		v, ok := server.MapMetric.(*server.MemStorage)
		if ok {
			b, err := json.Marshal(v)
			if err != nil {
				fmt.Println("Marshal block: " + err.Error())
				return
			}
			f.Write(b)
			fmt.Println("Metrics are saved!")
		} else {
			err := fmt.Errorf("passed wrong type of storage")
			panic(err)
		}
	}
}

func run(srv *http.Server) {
	server.ParseArgsServer()
	if *server.PsqlInfo != "" {
		err := server.InitDB()
		if err != nil {
			panic(err)
		}
	}
	srv.Addr = *server.HPServer
	fmt.Printf("Server is running on %s...\n", *server.HPServer)
	if *server.PsqlInfo != "" {
		defer func() {
			fmt.Println("Closing of db connection...")
			fmt.Println("Db is nil? - ", server.DB != nil)
			server.DB.Close()
		}()
	}
	if *server.PsqlInfo == "" {
		if *server.StartLoadMetric {
			b, err := os.ReadFile(*server.FSPath)
			if err == nil {
				err = json.Unmarshal(b, server.MapMetric)
				if err != nil {
					fmt.Println("Reading unmarshal block:", err.Error())
				} else {
					fmt.Println("Metrics are successfully loaded..")
				}
			}
		}
		go Storage()
	}
	err := srv.ListenAndServe()
	if err != http.ErrServerClosed {
		log.Fatalf("HTTP server ListenAndServe: %v", err)
	}
}
