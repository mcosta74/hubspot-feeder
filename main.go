package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/mcosta74/hubspot-feeder/internal/hubspotfeeder"
	"github.com/mcosta74/slogext"
	"github.com/oklog/run"
)

var (
	fs              *flag.FlagSet
	logLevel        slog.Level
	logUTC          bool
	logJSON         bool
	apiKey          string
	httpAddr        string
	pollInterval    time.Duration = 10 * time.Minute
	pollIntervalStr string
)

func init() {
	fs = flag.NewFlagSet("hubspot-feeder", flag.ExitOnError)
	fs.TextVar(&logLevel, "log.level", slog.LevelInfo, "the application log level")
	fs.BoolVar(&logUTC, "log.utc", true, "whether use UTC for log messages timestamp")
	fs.BoolVar(&logJSON, "log.json", false, "whether use JSON format for log messages")
	fs.StringVar(&apiKey, "api-key", "", "the HubSpot API Key")
	fs.StringVar(&httpAddr, "http.addr", ":8080", "Address for the HTTP server")
	fs.StringVar(&pollIntervalStr, "poll-interval", pollInterval.String(), "HubSpot poll interval")
}

func main() {
	_ = fs.Parse(os.Args[1:])

	logger := slogext.New(os.Stdout, slogext.WithLevel(logLevel), slogext.WithJSON(logJSON), slogext.WithUseUTC(logUTC))

	if apiKey == "" {
		logger.Error("missing API KEY")
		os.Exit(1)
	}

	pollInterval, err := time.ParseDuration(pollIntervalStr)
	if err != nil {
		logger.Error("bad format for poll-interval", "value", pollIntervalStr, "err", err)
		os.Exit(1)
	}

	logger.Info("service started")
	defer logger.Info("service stopped")

	var (
		repository = hubspotfeeder.NewRepository()
		poller     = hubspotfeeder.NewPoller(apiKey, logger.With("component", "POLLER"), repository, pollInterval)
		handler    = hubspotfeeder.MakeHttpHandler(repository)
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
	{
		httpLogger := logger.With("component", "HTTP")

		s := &http.Server{
			Addr:    httpAddr,
			Handler: handler,
		}

		g.Add(func() error {
			return s.ListenAndServe()
		}, func(err error) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if err := s.Shutdown(ctx); err != nil {
				httpLogger.Warn("failed to gracefully shutdown the server", "err", "err")
			} else {
				httpLogger.Debug("server gracefully shuted down")
			}
		})
	}

	logger.Info("service shutdown", "reason", g.Run())
}
