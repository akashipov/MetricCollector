package server

import (
	"flag"
	"github.com/caarlos0/env/v6"
	"log"
)

var HPServer *string

type ServerEnvConfig struct {
	Address string `env:"ADDRESS"`
}

func ParseArgsServer() {
	var cfg ServerEnvConfig
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
