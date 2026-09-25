package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yudai2929/codex-netmon/internal/netmon"
)

func main() {
	var config netmon.Config
	flag.StringVar(&config.Endpoint, "endpoint", "http://127.0.0.1:4318/v1/metrics", "OTLP/HTTP metrics endpoint")
	flag.DurationVar(&config.Interval, "interval", 5*time.Second, "sample and export interval")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := (netmon.Monitor{Config: config}).Run(ctx); err != nil {
		log.Fatal(err)
	}
}
