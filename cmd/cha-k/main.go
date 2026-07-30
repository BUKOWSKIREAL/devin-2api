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
	"path/filepath"
	"syscall"
	"time"

	"github.com/leookun/cha-k/internal/adapter"
	"github.com/leookun/cha-k/internal/adapter/devin"
	"github.com/leookun/cha-k/internal/app"
	"github.com/leookun/cha-k/internal/config"
	"github.com/leookun/cha-k/internal/debuglog"
)

func main() {
	configPath := flag.String("config", "config.yaml", "YAML 配置文件路径")
	flag.Parse()

	absoluteConfigPath, err := filepath.Abs(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	serviceConfig, err := config.Load(absoluteConfigPath)
	if err != nil {
		log.Fatal(err)
	}

	providerAdapter := adapter.Adapter(adapter.Unavailable{Reason: "provider adapter is not configured"})
	if serviceConfig.Devin.Token != "" {
		configured, createErr := devin.New(devin.Config{
			BaseURL: serviceConfig.Devin.BaseURL,
			Token:   serviceConfig.Devin.Token,
			Model:   serviceConfig.Devin.Model,
		})
		if createErr != nil {
			log.Fatal(createErr)
		}
		providerAdapter = configured
	}
	debugManager := debuglog.NewManager(filepath.Join(filepath.Dir(absoluteConfigPath), "logs"))
	application := app.New(providerAdapter, serviceConfig.Server, debugManager)
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
