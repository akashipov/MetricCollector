package internal

import (
	"errors"
	"flag"
	"fmt"
	"github.com/caarlos0/env/v6"
	"log"
	"strings"
)

var HPClient *string
var HPServer *string
var ReportInterval *int
var PollInterval *int

type ClientEnvConfig struct {
	Address        string `env:"ADDRESS"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
	PollInterval   int    `env:"POLL_INTERVAL"`
}

type ServerEnvConfig struct {
	Address string `env:"ADDRESS"`
}

func ParseArgsClient() {
	var cfg ClientEnvConfig
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal(err)
	}
	HPClient = flag.String("a", ":8080", "host and port in format <host>:<port>")

	ReportInterval = flag.Int(
		"r", 10, "period of time in seconds, throw of it will be report to the server",
	)
	PollInterval = flag.Int(
		"p", 2, "period of time in seconds, throw of it metrics will be updated from 'runtime'",
	)
	flag.Parse()
	if cfg.Address != "" {
		sep := ":"
		if !strings.Contains(cfg.Address, sep) {
			panic(errors.New(fmt.Sprintf("ADDRESS should contain %s symbol to separate host and port", sep)))
		}
		HPClient = &cfg.Address
	}
	if cfg.ReportInterval != 0 {
		ReportInterval = &cfg.ReportInterval
	}
	if cfg.PollInterval != 0 {
		PollInterval = &cfg.PollInterval
	}
}

func ParseArgsServer() {
	var cfg ClientEnvConfig
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal(err)
	}
	HPServer = flag.String("a", ":8080", "host and port in format <host>:<port>")
	flag.Parse()
	if cfg.Address != "" {
		HPServer = &cfg.Address
	}
}
