package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"syscall"

	"github.com/mcosta74/hubspot-feeder/internal/hubspotfeeder"
	"github.com/mcosta74/slogext"
	"github.com/oklog/run"
)

var (
	fs *flag.FlagSet

	logLevel slog.Level

	apiKey string
)

func init() {
	fs = flag.NewFlagSet("hubspot-feeder", flag.ExitOnError)

	fs.TextVar(&logLevel, "log.level", slog.LevelInfo, "the application log level")

	fs.StringVar(&apiKey, "api-key", "", "the HibSpot API Key")
}

func main() {
	_ = fs.Parse(os.Args[1:])

	logger := slogext.New(os.Stdout, slogext.WithLevel(logLevel))

	if apiKey == "" {
		logger.Error("missing API KEY")
		os.Exit(1)
	}

	logger.Info("service started")
	defer logger.Info("service stopped")

	var (
		poller = hubspotfeeder.NewPoller(apiKey, logger.With("component", "POLLER"))
	)

	var g run.Group
	{
		// Signal Handler
		g.Add(run.SignalHandler(context.Background(), syscall.SIGINT, syscall.SIGTERM))
	}
	{
		ctx, cancel := context.WithCancel(context.Background())

		// Poller
		g.Add(func() error {
			return poller.Poll(ctx)
		}, func(err error) {
			cancel()
		})

	}

	logger.Info("service shutdown", "reason", g.Run())
}
