// 本文件负责加载配置、组装服务依赖并启动 HTTP 服务器。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/leookun/cha-k/internal/adapter"
	"github.com/leookun/cha-k/internal/app"
	"github.com/leookun/cha-k/internal/config"
)

func main() {
	configPath := flag.String("config", "config.yaml", "YAML 配置文件路径")
	flag.Parse()

	serviceConfig, err := config.Load(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	application := app.New(adapter.Unavailable{Reason: "provider adapter is not configured"}, serviceConfig.Server)
	server := application.HTTPServer()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, server); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, server interface {
	ListenAndServe() error
	Shutdown(context.Context) error
}) error {
	result := make(chan error, 1)
	go func() {
		result <- server.ListenAndServe()
	}()

	select {
	case err := <-result:
		if errors.Is(err, http.ErrServerClosed) || errors.Is(err, context.Canceled) {
			return nil
		}
		return fmt.Errorf("serve HTTP: %w", err)
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown HTTP server: %w", err)
	}
	return nil
}
