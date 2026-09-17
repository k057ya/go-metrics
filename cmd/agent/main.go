package main

import (
	"flag"
	"fmt"
	"log"
	"strconv"

	"github.com/caarlos0/env/v6"
	"github.com/k057ya/go-metrics/internal/agent"
	"github.com/k057ya/go-metrics/internal/config"
)

type EnvConfig struct {
	Address        string `env:"ADDRESS"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
	PollInterval   int    `env:"POLL_INTERVAL"`
}

func main() {

	flag.Var(config.ClientConfig.Server, "a", "Server host and port")
	flag.Var(config.ClientConfig.ReportInterval, "r", "Report sending interval in seconds")
	flag.Var(config.ClientConfig.PollInterval, "p", "Fetch metrics poll interval in seconds")
	flag.Parse()

	var cfg EnvConfig
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	if cfg.Address != "" {
		err := config.ClientConfig.Server.Set(cfg.Address)
		if err != nil {
			fmt.Println(err, ", falling back to", config.ClientConfig.Server.String())
		}
	}

	if cfg.ReportInterval != 0 {
		err := config.ClientConfig.ReportInterval.Set(strconv.Itoa(cfg.ReportInterval))
		if err != nil {
			fmt.Println(err, ", falling back to", config.ClientConfig.ReportInterval.String())
		}
	}

	if cfg.PollInterval != 0 {
		err := config.ClientConfig.ReportInterval.Set(strconv.Itoa(cfg.PollInterval))
		if err != nil {
			fmt.Println(err, ", falling back to", config.ClientConfig.PollInterval.String())
		}
	}

	fmt.Printf("Starting agent with params: server_addr=%s, report_interval=%s, poll_interval=%s\n",
		config.ClientConfig.Server.Address(),
		config.ClientConfig.ReportInterval.String(),
		config.ClientConfig.PollInterval.String(),
	)

	var a = agent.Agent{
		Client:         agent.NewHTTPClient(config.ClientConfig.Server.Address()),
		PollInterval:   config.ClientConfig.PollInterval.Interval,
		ReportInterval: config.ClientConfig.ReportInterval.Interval,
	}

	err = agent.Run(a)
	fmt.Println("Error starting agent: " + err.Error())
}
