package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/framjet/go-webhook-middleman/internal/cliutil"
	"github.com/framjet/go-webhook-middleman/internal/server"
	"github.com/urfave/cli/v3"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var (
	Version   = "DEV"
	BuildTime = "unknown"
	BuildType = ""
	Runtime   = "host"
)

func setupLogger(level string, jsonFormat bool) *slog.Logger {
	var logLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	var handler slog.Handler
	if jsonFormat {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}

func runServer(ctx context.Context, c *cli.Command) error {
	host := c.String("host")
	port := c.String("port")
	configPath := c.String("config")
	logLevel := c.String("log-level")
	jsonFormat := c.Bool("json-log")
	timeout := c.Duration("timeout")

	logger := setupLogger(logLevel, jsonFormat)

	// Validate config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		logger.Error("Configuration file not found", "path", configPath)
		return fmt.Errorf("configuration file not found: %s", configPath)
	}

	srv, err := server.NewWebhookServer(configPath, timeout, logger)
	if err != nil {
		logger.Error("Failed to create webhook srv", "error", err)
		return err
	}

	// Setup router with dynamic routes
	router := srv.SetupRoutes()

	// Create HTTP srv
	addr := fmt.Sprintf("%s:%s", host, port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: timeout + 10*time.Second, // Allow extra time for forwarding
		IdleTimeout:  120 * time.Second,
	}

	// Start srv in goroutine
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("Starting webhook middleman server",
			"host", host,
			"port", port,
			"config", configPath,
			"destinations", len(srv.Config.Destinations),
			"routes", len(srv.Config.Routes),
			"timeout", timeout,
			"json_log", jsonFormat)

		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// Setup graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		logger.Info("Shutting down srv...")
	case err := <-serverErr:
		logger.Error("Server error", "error", err)
		return err
	}

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
		return err
	}

	logger.Info("Server stopped")
	return nil
}

func main() {
	bInfo := cliutil.GetBuildInfo(BuildType, Version)

	cmd := &cli.Command{
		Name:      "FramJet WebHook Router Middleman Server",
		Usage:     "A simple HTTP server that routes webhook calls to multiple destinations based on YAML configuration.",
		UsageText: "framjet-webhook-middleman [global options] [command options]",
		Version:   fmt.Sprintf("%s (built %s%s)", Version, BuildTime, bInfo.GetBuildTypeMsg()),
		Copyright: fmt.Sprintf(
			`(c) %d FramJet.
   Your installation of this software constitutes a symbol of your signature indicating that you accept
   the terms of the MIT (https://github.com/framjet/go-webhook-middleman?tab=MIT-1-ov-file).`, time.Now().Year(),
		),
		EnableShellCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "host",
				Aliases: []string{"s"},
				Value:   "0.0.0.0",
				Usage:   "Host to bind to",
				Sources: cli.EnvVars("HTTP_HOST"),
			},
			&cli.StringFlag{
				Name:    "port",
				Aliases: []string{"p"},
				Value:   "8080",
				Usage:   "Port to listen on",
				Sources: cli.EnvVars("HTTP_PORT"),
			},
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Value:   "config.yaml",
				Usage:   "Path to configuration file",
				Sources: cli.EnvVars("CONFIG_FILE"),
			},
			&cli.StringFlag{
				Name:    "log-level",
				Aliases: []string{"l"},
				Value:   "info",
				Usage:   "Log level (debug, info, warn, error)",
				Sources: cli.EnvVars("LOG_LEVEL"),
			},
			&cli.BoolFlag{
				Name:    "json-log",
				Aliases: []string{"j"},
				Value:   false,
				Usage:   "Enable JSON formatted logging",
				Sources: cli.EnvVars("JSON_LOG"),
			},
			&cli.DurationFlag{
				Name:    "timeout",
				Aliases: []string{"t"},
				Value:   30 * time.Second,
				Usage:   "HTTP client timeout",
				Sources: cli.EnvVars("HTTP_TIMEOUT"),
			},
		},
		Action: runServer,
		Commands: []*cli.Command{
			{
				Name:  "version",
				Usage: "Print version information",
				Action: func(ctx context.Context, c *cli.Command) error {
					fmt.Printf("framjet-webhook-middleman version %s\n", c.Version)
					return nil
				},
			},
		},
		Description: `Webhook Middleman Server routes single webhook calls to multiple destinations based on YAML configuration.

The server accepts requests to configurable paths and forwards them to configured destinations based on matching rules.

Configuration file format:
  destinations:
    discord: "https://discord.com/api/webhooks/..."
    slack: 
      url: "https://hooks.slack.com/services/..."
      method: "PUT"
      body: |
        {
          "text": "{{.params.service}} deployed to {{.var.environment}}"
        }
  
  variables:
    environment: "production"
  
  routes:
    - method: "POST"
      path: "/{service}/{event}"
      matchers:
        - service: "frontend"
          event: "deployment"
          to: ["discord", "slack"]
        - service: "/api-.+/"
          to:
            - discord
            - url: "https://custom.webhook.com/{{.params.service}}"
              method: "PUT"
              body: '{"event": "{{.params.event}}"}'
    
    - path: "/{app}/{env}/{version}"
      matchers:
        - app: ["frontend", "backend"]
          env: "prod"
          to: ["slack"]

Matching rules:
  - Exact string: "frontend"  
  - Regex: "/api-.+/"
  - Arrays: ["frontend", "backend", "/api-.+/"]]`,
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
