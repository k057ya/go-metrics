package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/caarlos0/env/v11"
	"github.com/k057ya/go-metrics/internal/agent"
	"github.com/k057ya/go-metrics/internal/config"
	"github.com/k057ya/go-metrics/internal/logger"
	"go.uber.org/zap"
)

type EnvConfig struct {
	Address        string `env:"ADDRESS"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
	PollInterval   int    `env:"POLL_INTERVAL"`
	SignatureKey   string `env:"KEY"`
	RateLimit      int    `env:"RATE_LIMIT"`
}

func main() {
	// Init logger
	if err := logger.Initialize("info"); err != nil {
		fmt.Fprintln(os.Stderr, "Error while initializing logger:", err)
		return
	}
	defer logger.Log.Sync()

	// Prepare config
	parseFlags()
	if err := parseEnv(); err != nil {
		logger.Log.Warn("Unable to parse ENV-vars, falling back to flag values", zap.Error(err))
	}

	logger.Log.Sugar().Infof(
		"Starting agent with params: server_addr=%s, report_interval=%s, poll_interval=%s",
		config.ClientConfig.Server.Address(),
		config.ClientConfig.ReportInterval.String(),
		config.ClientConfig.PollInterval.String(),
	)

	var a = agent.Agent{
		Client: agent.NewHTTPClient(
			config.ClientConfig.Server.Address(),
			config.ClientConfig.Signature.Key,
		),
		PollInterval:   config.ClientConfig.PollInterval.Interval,
		ReportInterval: config.ClientConfig.ReportInterval.Interval,
		WorkerCount:    config.ClientConfig.WorkerCount,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := agent.Run(ctx, a); err != nil {
		logger.Log.Error("agent run error", zap.Error(err))
	}
}

func parseEnv() error {
	var cfg EnvConfig
	err := env.Parse(&cfg)
	if err != nil {
		return err
	}

	setConf := func(target flag.Value, value string) {
		err := target.Set(value)
		if err != nil {
			logger.Log.Sugar().Warnf("%v falling back to %s", err, target.String())
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

	if cfg.SignatureKey != "" {
		config.ClientConfig.Signature.Key = cfg.SignatureKey
	}

	if cfg.RateLimit > 0 {
		config.ClientConfig.WorkerCount = cfg.RateLimit
	}
	return err
}

func parseFlags() {
	flag.Var(config.ClientConfig.Server, "a", "Server host and port")
	flag.Var(config.ClientConfig.ReportInterval, "r", "Report sending interval in seconds")
	flag.Var(config.ClientConfig.PollInterval, "p", "Fetch metrics poll interval in seconds")
	flag.StringVar(&config.ClientConfig.Signature.Key, "k", "", "Key for signing requests")
	flag.IntVar(&config.ClientConfig.WorkerCount, "l", config.DefaultWorkerCount, "Rate limit for sending requests")

	flag.Parse()
}
