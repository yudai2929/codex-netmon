package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yudai2929/codex-netmon/internal/localstack"
	"github.com/yudai2929/codex-netmon/internal/netmon"
)

const defaultEndpoint = "http://127.0.0.1:4318/v1/metrics"

func main() {
	var config netmon.Config
	var startLocalStack bool
	var grafanaPort, otlpPort int
	flag.StringVar(&config.Endpoint, "endpoint", defaultEndpoint, "OTLP/HTTP metrics endpoint")
	flag.DurationVar(&config.Interval, "interval", 5*time.Second, "sample and export interval")
	flag.BoolVar(&startLocalStack, "local-stack", false, "start the local LGTM stack and print the dashboard URL")
	flag.IntVar(&grafanaPort, "grafana-port", 3000, "localhost port for Grafana when using -local-stack")
	flag.IntVar(&otlpPort, "otlp-port", 4318, "localhost OTLP/HTTP port when using -local-stack")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if startLocalStack {
		info, err := (localstack.Stack{Config: localstack.Config{GrafanaPort: grafanaPort, OTLPPort: otlpPort}}).Start(ctx, localStackAssets, os.Stdout, os.Stderr)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintln(os.Stdout, "Local LGTM stack started.")
		fmt.Fprintf(os.Stdout, "Grafana dashboard: %s\n", info.DashboardURL)
		fmt.Fprintf(os.Stdout, "OTLP metrics endpoint: %s\n", info.OTLPEndpoint)
		fmt.Fprintf(os.Stdout, "Stop the stack: %s\n", info.StopCommand)
		if !flagWasSet("endpoint") {
			config.Endpoint = info.OTLPEndpoint
		}
	}

	if err := (netmon.Monitor{Config: config}).Run(ctx); err != nil {
		log.Fatal(err)
	}
}

func flagWasSet(name string) bool {
	set := false
	flag.Visit(func(item *flag.Flag) {
		if item.Name == name {
			set = true
		}
	})
	return set
}
