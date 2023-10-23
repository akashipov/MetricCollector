package main

import (
	"context"
	"fmt"
	"github.com/akashipov/MetricCollector/internal/server"
	"go.uber.org/zap"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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

func run(srv *http.Server) {
	server.ParseArgsServer()
	srv.Addr = *server.HPServer
	fmt.Printf("Server is running on %s...\n", *server.HPServer)
	err := srv.ListenAndServe()
	if err != http.ErrServerClosed {
		log.Fatalf("HTTP server ListenAndServe: %v", err)
	}
}
