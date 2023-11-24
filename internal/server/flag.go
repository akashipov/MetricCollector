package server

import (
	"flag"
	"fmt"
	"log"

	"github.com/caarlos0/env/v6"
)

var HPServer *string
var PTSave *int
var FSPath *string
var StartLoadMetric *bool
var ServerKey *string
var PsqlInfo *string

type ServerEnvConfig struct {
	Address            string  `env:"ADDRESS"`
	StoreInterval      *int    `env:"STORE_INTERVAL"`
	FileStoragePath    *string `env:"FILE_STORAGE_PATH"`
	StartLoadMetric    *bool   `env:"RESTORE"`
	ConnectionDBString *string `env:"DATABASE_DSN"`
	KeyForHash         *string `env:"KEY"`
}

func ParseArgsServer() {
	var cfg ServerEnvConfig
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal(err)
	}
	HPServer = flag.String("a", ":8080", "host and port in format <host>:<port>")
	PTSave = flag.Int("i", 300, "interval to save metrics data to file in seconds")
	FSPath = flag.String("f", "/tmp/metrics-db.json", "File storage path to json")
	PsqlInfo = flag.String(
		"d",
		"", // "host=localhost port=5432 user=postgres password=620631 dbname=metrics sslmode=disable",
		"Connection string for psql",
	)
	StartLoadMetric = flag.Bool("r", true, "Either load last metric checkpoint")
	ServerKey = flag.String("k", "", "Key to create hash and check sign")
	flag.Parse()
	if cfg.Address != "" {
		HPServer = &cfg.Address
	}
	if cfg.StoreInterval != nil {
		PTSave = cfg.StoreInterval
	}
	if cfg.FileStoragePath != nil {
		FSPath = cfg.FileStoragePath
	}
	if cfg.StartLoadMetric != nil {
		StartLoadMetric = cfg.StartLoadMetric
	}
	if cfg.KeyForHash != nil {
		ServerKey = cfg.KeyForHash
	}
	if cfg.ConnectionDBString != nil {
		PsqlInfo = cfg.ConnectionDBString
	}
	fmt.Println("Connection string for psql:", *PsqlInfo)
	fmt.Println("StartLoadMetric:", *StartLoadMetric)
	fmt.Println("Path for metrics file:", *FSPath)
	fmt.Println("Interval to save metrics:", *PTSave)
}
