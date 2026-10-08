package main

import (
	"flag"
	"fmt"
	"log"
	"strconv"

	"github.com/caarlos0/env/v11"
	"github.com/k057ya/go-metrics/internal/agent"
	"github.com/k057ya/go-metrics/internal/config"
)

type EnvConfig struct {
	Address        string `env:"ADDRESS"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
	PollInterval   int    `env:"POLL_INTERVAL"`
}

func main() {

	// Prepare config
	parseFlags()
	if err := parseEnv(); err != nil {
		fmt.Printf("Unable to parse ENV-vars, falling back to flag values: %s\n", err)
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

	err := agent.Run(a)
	fmt.Println("Error starting agent: " + err.Error())
}

func parseEnv() error {
	var cfg EnvConfig
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal(err)
		return err
	}

	setConf := func(target flag.Value, value string) {
		err := target.Set(value)
		if err != nil {
			fmt.Printf("%v falling back to %s\n", err, target.String())
		}
	}

	if cfg.Address != "" {
		setConf(config.ClientConfig.Server, cfg.Address)
	}

	if cfg.ReportInterval != 0 {
		setConf(config.ClientConfig.ReportInterval, strconv.Itoa(cfg.ReportInterval))
	}

	if cfg.PollInterval != 0 {
		setConf(config.ClientConfig.PollInterval, strconv.Itoa(cfg.PollInterval))
	}
	return err
}

func parseFlags() {
	flag.Var(config.ClientConfig.Server, "a", "Server host and port")
	flag.Var(config.ClientConfig.ReportInterval, "r", "Report sending interval in seconds")
	flag.Var(config.ClientConfig.PollInterval, "p", "Fetch metrics poll interval in seconds")
	flag.Parse()
}
