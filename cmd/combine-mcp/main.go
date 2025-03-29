package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/nazar256/combine-mcp/pkg/aggregator"
	"github.com/nazar256/combine-mcp/pkg/config"
	"github.com/nazar256/combine-mcp/pkg/logger"
	"github.com/nazar256/combine-mcp/pkg/stdio"
)

const (
	// Version is the version of the MCP aggregator
	Version = "1.0.0"
	// Name is the name of the MCP aggregator
	Name = "mcp-aggregator"
)

func main() {
	// SET UP STDOUT REDIRECTION FIRST - before anything else!
	// We need to capture ALL stdout output and redirect it

	// Create a pipe for capturing stdout
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating stdout pipe: %v\n", err)
		os.Exit(1)
	}

	// Save the original stdout file descriptor
	oldStdoutFd := int(os.Stdout.Fd())
	// Get a duplicate file descriptor for the original stdout
	realStdoutFd, err := syscall.Dup(oldStdoutFd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error duplicating stdout fd: %v\n", err)
		os.Exit(1)
	}

	// Create a new file from the duplicated fd that we'll use later
	realStdout := os.NewFile(uintptr(realStdoutFd), "stdout")

	// Now replace stdout with our pipe writer
	os.Stdout = stdoutWriter

	// Start a goroutine to read from the pipe and redirect to stderr
	// This ensures ANY fmt.Printf or println from any library gets redirected
	go func() {
		// We want to read stdout continuously
		buffer := make([]byte, 4096)
		for {
			n, err := stdoutReader.Read(buffer)
			if err != nil {
				if err != io.EOF {
					fmt.Fprintf(os.Stderr, "Error reading from stdout pipe: %v\n", err)
				}
				break
			}
			if n > 0 {
				// Write the captured output to stderr instead
				os.Stderr.Write(buffer[:n])
			}
		}
	}()

	// Create a context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Load configuration
	cfg, err := config.LoadConfig("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize the logger
	if err := logger.Init(cfg.LogLevel, cfg.LogFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()

	// Log startup message to file only
	logger.Info("Starting MCP Aggregator v%s", Version)
	logger.Debug("Configuration loaded: %d servers configured", len(cfg.Servers))

	// Only print startup messages to stderr, never stdout
	fmt.Fprintf(os.Stderr, "Starting MCP Aggregator v%s\n", Version)

	// Create and initialize the aggregator
	agg := aggregator.NewMCPAggregator()
	if err := agg.Initialize(ctx, cfg); err != nil {
		logger.Fatal("Error initializing aggregator: %v", err)
	}
	defer agg.Close()

	// Create the MCP server
	server := stdio.NewAggregatorServer(Name, Version, agg)

	// Register tools from the aggregator
	if err := server.RegisterTools(); err != nil {
		logger.Fatal("Error registering tools: %v", err)
	}

	// Start the server - logging to file only
	logger.Debug("Starting stdio server")
	fmt.Fprintf(os.Stderr, "Server started, listening on stdin/stdout\n")

	// Close the writer to stop the redirection goroutine
	// This ensures we've processed all previous stdout writes before we restore
	stdoutWriter.Close()

	// Restore stdout to the original file we captured earlier
	os.Stdout = realStdout

	// Clean up when done
	defer realStdout.Close()

	// Now serve using our clean stdout
	if err := server.ServeStdio(); err != nil {
		logger.Fatal("Error serving MCP: %v", err)
	}
}
