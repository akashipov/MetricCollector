package agent

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/caarlos0/env/v6"
)

var HPClient *string
var ReportInterval *int
var PollInterval *int
var AgentKey *string
var RateLimit *int

type ClientEnvConfig struct {
	Address        *string `env:"ADDRESS"`
	ReportInterval *int    `env:"REPORT_INTERVAL"`
	PollInterval   *int    `env:"POLL_INTERVAL"`
	KeyForHash     *string `env:"KEY"`
	RateLimit      *int    `env:"RATE_LIMIT"`
}

func ParseArgsClient() {
	var cfg ClientEnvConfig
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal(err)
	}
	HPClient = flag.String("a", "localhost:8080", "host and port in format <host>:<port>")

	ReportInterval = flag.Int(
		"r", 10, "period of time in seconds, throw of it will be report to the server",
	)
	PollInterval = flag.Int(
		"p", 2, "period of time in seconds, throw of it metrics will be updated from 'runtime'",
	)
	AgentKey = flag.String(
		"k", "", "Key to hash requsts and check the sign from server",
	)
	RateLimit = flag.Int(
		"l", 1, "Limit of simulteniously sending of requests to server",
	)
	flag.Parse()
	if cfg.Address != nil {
		sep := ":"
		if !strings.Contains(*cfg.Address, sep) {
			panic(fmt.Errorf("ADDRESS should contain %s symbol to separate host and port", sep))
		}
		HPClient = cfg.Address
	}
	if cfg.ReportInterval != nil {
		ReportInterval = cfg.ReportInterval
	}
	if cfg.PollInterval != nil {
		PollInterval = cfg.PollInterval
	}
	if cfg.KeyForHash != nil {
		AgentKey = cfg.KeyForHash
	}
	if cfg.RateLimit != nil {
		RateLimit = cfg.RateLimit
	}
	fmt.Printf("Poll interval size is %d seconds\n", *PollInterval)
	fmt.Printf("Report interval size is %d seconds\n", *ReportInterval)
	fmt.Printf("Rate limit is %d\n", *RateLimit)
}
