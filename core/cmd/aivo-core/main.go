package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	coreapp "aivo/core/app"
	"aivo/core/domain"
	"aivo/core/infra/persistence"
	corehttp "aivo/core/internal/transport/http"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "provider-smoke" {
		os.Exit(runProviderSmokeCommand(context.Background(), os.Args[2:], os.Stdout, os.Stderr))
	}
	if len(os.Args) > 1 && (os.Args[1] == "-h" || os.Args[1] == "--help" || os.Args[1] == "help") {
		printUsage(os.Stdout)
		return
	}
	runServer()
}

func runServer() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := corehttp.NewRouter()
	server := &http.Server{
		Addr:              coreServerAddr(),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		logger.Error("server failed", "error", err)
		os.Exit(1)
	}
	server.Addr = listener.Addr().String()
	if strings.TrimSpace(os.Getenv("AIVO_CORE_READY_STDOUT")) == "1" {
		if err := writeCoreReadyRecord(os.Stdout, server.Addr); err != nil {
			_ = listener.Close()
			logger.Error("core readiness announcement failed", "error", err)
			os.Exit(1)
		}
	}

	go func() {
		logger.Info("aivo core listening", "addr", server.Addr)
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if shutdowner, ok := handler.(interface{ Shutdown() }); ok {
		shutdowner.Shutdown()
	}
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("shutdown failed", "error", err)
		os.Exit(1)
	}
}

func coreServerAddr() string {
	if addr := strings.TrimSpace(os.Getenv("AIVO_CORE_ADDR")); addr != "" {
		return addr
	}
	return "127.0.0.1:43117"
}

const coreReadyPrefix = "AIVO_CORE_READY "

type coreReadyRecord struct {
	Version int    `json:"version"`
	URL     string `json:"url"`
}

func writeCoreReadyRecord(out io.Writer, addr string) error {
	host, rawPort, err := net.SplitHostPort(strings.TrimSpace(addr))
	if err != nil {
		return fmt.Errorf("parse listener address: %w", err)
	}
	port, err := strconv.Atoi(rawPort)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("listener port must be non-zero and valid")
	}
	if host != "127.0.0.1" {
		return fmt.Errorf("listener host must be the IPv4 loopback address")
	}
	endpoint := (&url.URL{Scheme: "http", Host: net.JoinHostPort(host, rawPort)}).String()
	payload, err := json.Marshal(coreReadyRecord{Version: 1, URL: endpoint})
	if err != nil {
		return fmt.Errorf("encode readiness record: %w", err)
	}
	if _, err := fmt.Fprintf(out, "%s%s\n", coreReadyPrefix, payload); err != nil {
		return fmt.Errorf("write readiness record: %w", err)
	}
	return nil
}

func runProviderSmokeCommand(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("provider-smoke", flag.ContinueOnError)
	flags.SetOutput(stderr)
	providerID := flags.String("provider", "", "provider id to check")
	modelID := flags.String("model", "", "model id to check")
	includeModels := flags.Bool("include-model-list", false, "include full model list in the JSON result")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *providerID == "" {
		_, _ = fmt.Fprintln(stderr, "provider-smoke requires --provider")
		return 2
	}
	store, err := persistence.OpenDefault()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "open store: %v\n", err)
		return 1
	}
	defer store.Close()
	service := coreapp.NewService(store)
	return runProviderSmoke(ctx, service, domain.ProviderIntegrationCheckInput{
		ProviderID:       *providerID,
		ModelID:          *modelID,
		IncludeModelList: *includeModels,
	}, stdout, stderr)
}

func runProviderSmoke(ctx context.Context, service *coreapp.Service, input domain.ProviderIntegrationCheckInput, stdout io.Writer, stderr io.Writer) int {
	result, err := service.CheckProviderIntegration(ctx, input)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "provider smoke failed: %v\n", err)
		return 1
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		_, _ = fmt.Fprintf(stderr, "write result: %v\n", err)
		return 1
	}
	if !result.Ready {
		return 1
	}
	return 0
}

func printUsage(out io.Writer) {
	_, _ = fmt.Fprintln(out, "Usage:")
	_, _ = fmt.Fprintln(out, "  aivo-core")
	_, _ = fmt.Fprintln(out, "  aivo-core provider-smoke --provider <id> [--model <id>] [--include-model-list]")
}
