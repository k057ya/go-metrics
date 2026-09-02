package main

import (
	"flag"
	"fmt"

	internal_agent "github.com/k057ya/go-metrics/internal/agent"
	"github.com/k057ya/go-metrics/internal/config"
)

func main() {

	flag.Var(config.ClientConfig.Server, "a", "Server host and port")
	flag.Var(config.ClientConfig.ReportInterval, "r", "Report sending interval in seconds")
	flag.Var(config.ClientConfig.PollInterval, "p", "Fetch metrics poll interval in seconds")
	flag.Parse()

	fmt.Printf("Running agent with params: server_addr=%s, report_interval=%s, poll_interval=%s",
		config.ClientConfig.Server.Address(),
		config.ClientConfig.ReportInterval.String(),
		config.ClientConfig.PollInterval.String(),
	)

	internal_agent.Run()
}
