package main

import (
	"flag"
	"fmt"

	"github.com/k057ya/go-metrics/internal/agent"
	"github.com/k057ya/go-metrics/internal/config"
)

func main() {

	flag.Var(config.ClientConfig.Server, "a", "Server host and port")
	flag.Var(config.ClientConfig.ReportInterval, "r", "Report sending interval in seconds")
	flag.Var(config.ClientConfig.PollInterval, "p", "Fetch metrics poll interval in seconds")
	flag.Parse()

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
