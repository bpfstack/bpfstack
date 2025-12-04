package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bpfstack/bpfstack/internal/config"
	"github.com/bpfstack/bpfstack/pkg/action"
	"github.com/bpfstack/bpfstack/pkg/action/compute"
	"github.com/bpfstack/bpfstack/pkg/logger"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	logFormat := flag.String("log-format", "text", "Log format (text, json)")
	flag.Parse()

	// Configure global logger
	cfg := &logger.Config{
		Level:  parseLogLevel(*logLevel),
		Format: parseLogFormat(*logFormat),
		Output: os.Stdout,
	}
	logger.SetGlobalConfig(cfg)

	// Create main logger
	log := logger.New("main")

	log.Info("BPFStack starting", logger.Fields{
		"config":     *configPath,
		"log_level":  *logLevel,
		"log_format": *logFormat,
	})

	// Check if config file exists
	if _, err := os.Stat(*configPath); os.IsNotExist(err) {
		log.Error("config file not found", logger.Fields{
			"path": *configPath,
		})
		os.Exit(1)
	}

	// Create action registry and register actions
	registry := action.NewRegistry()

	// Register compute actions (cpu_metrics)
	compute.RegisterActions(registry)

	log.Info("registered actions", logger.Fields{
		"actions": strings.Join(registry.Names(), ", "),
	})

	// Create action manager
	manager := action.NewManager(registry)

	// Create config watcher for hot-reloading
	watcher, err := config.NewConfigWatcher(*configPath)
	if err != nil {
		log.Error("failed to create config watcher", logger.Fields{
			"error": err.Error(),
		})
		os.Exit(1)
	}

	// Start the manager with the watcher
	if err := manager.StartWithWatcher(watcher); err != nil {
		log.Error("failed to start manager", logger.Fields{
			"error": err.Error(),
		})
		os.Exit(1)
	}

	log.Info("BPFstack is running")
	log.Info("modify the config file to enable/disable actions in real-time")

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println() // New line after ^C
	log.Info("shutting down")

	// Stop all actions
	manager.StopAll()

	log.Info("BPFStack stopped")
}

func parseLogLevel(level string) logger.Level {
	switch strings.ToLower(level) {
	case "debug":
		return logger.DEBUG
	case "info":
		return logger.INFO
	case "warn", "warning":
		return logger.WARN
	case "error":
		return logger.ERROR
	default:
		return logger.INFO
	}
}

func parseLogFormat(format string) logger.Format {
	switch strings.ToLower(format) {
	case "json":
		return logger.FormatJSON
	default:
		return logger.FormatText
	}
}
