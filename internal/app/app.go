// 本文件定义 HTTP 应用、chi 路由和供应商适配器的串联逻辑。
//
// Package app 负责组装 HTTP 路由并连接 API 编解码与供应商适配器。
package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/leookun/cha-k/internal/adapter"
	"github.com/leookun/cha-k/internal/api/openai/responses"
	"github.com/leookun/cha-k/internal/config"
	"github.com/leookun/cha-k/internal/llm"
)

const (
	// readHeaderTimeout 是防止慢速请求头连接长期占用资源的内部策略。
	readHeaderTimeout = 60 * time.Second
	// idleTimeout 是 keep-alive 连接两次请求之间的内部空闲策略。
	idleTimeout = 360 * time.Second
)

// App 保存 HTTP 应用依赖和服务配置。
type App struct {
	// adapter 是供应商无关请求与上游协议之间的适配器。
	adapter adapter.Adapter
	// serverConfig 是 HTTP 服务运行配置。
	serverConfig config.ServerConfig
}

// New 创建一个使用指定供应商适配器的 HTTP 应用。
func New(providerAdapter adapter.Adapter, serverConfig config.ServerConfig) *App {
	return &App{adapter: providerAdapter, serverConfig: serverConfig}
}

// Router 返回应用的 chi HTTP 路由。
func (application *App) Router() http.Handler {
	router := chi.NewRouter()
	router.Get("/healthz", application.health)
	router.Post("/v1/responses", application.createResponses)
	return router
}

// HTTPServer 创建带有应用路由和超时配置的 HTTP 服务。
func (application *App) HTTPServer() *http.Server {
	return &http.Server{
		Addr:              application.serverConfig.Listen,
		Handler:           application.Router(),
		ReadHeaderTimeout: readHeaderTimeout,
		IdleTimeout:       idleTimeout,
	}
}

func (application *App) health(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	_, _ = writer.Write([]byte(`{"status":"ok"}` + "\n"))
}

func (application *App) createResponses(writer http.ResponseWriter, request *http.Request) {
	if application.adapter == nil {
		writeError(writer, http.StatusServiceUnavailable, "provider adapter is not configured")
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(writer, request.Body, 8<<20))
	if err != nil {
		writeError(writer, http.StatusBadRequest, fmt.Sprintf("read request: %v", err))
		return
	}
	adapted, err := responses.DecodeRequest(body)
	if err != nil {
		writeError(writer, http.StatusBadRequest, err.Error())
		return
	}
	stream, err := application.adapter.Stream(request.Context(), adapted.Context)
	if err != nil {
		writeError(writer, http.StatusBadGateway, err.Error())
		return
	}
	if adapted.Options.Stream {
		if err := writeSSE(request.Context(), writer, stream); err != nil {
			return
		}
		return
	}
	message, err := collectFinalMessage(request.Context(), stream)
	if err != nil {
		writeError(writer, http.StatusBadGateway, err.Error())
		return
	}
	body, err = responses.EncodeResponse(message)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, err.Error())
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	_, _ = writer.Write(body)
}

func writeSSE(ctx context.Context, writer http.ResponseWriter, stream llm.ResponseStream) error {
	flusher, ok := writer.(http.Flusher)
	if !ok {
		return errors.New("streaming response writer does not support flushing")
	}
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("Connection", "keep-alive")
	for {
		event, err := stream.Recv(ctx)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		encoded, err := responses.EncodeEvent(event)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(writer, "event: %s\ndata: %s\n\n", encoded.Name, encoded.Data); err != nil {
			return err
		}
		flusher.Flush()
	}
}

func collectFinalMessage(ctx context.Context, stream llm.ResponseStream) (*llm.AssistantMessage, error) {
	var final *llm.AssistantMessage
	for {
		event, err := stream.Recv(ctx)
		if errors.Is(err, io.EOF) {
			if final == nil {
				return nil, errors.New("response stream ended without a final message")
			}
			return final, nil
		}
		if err != nil {
			return nil, err
		}
		switch event.Type {
		case llm.ResponseEventDone:
			if event.Message == nil {
				return nil, errors.New("done event has no final message")
			}
			final = event.Message
		case llm.ResponseEventError:
			if event.Error == nil {
				return nil, errors.New("error event has no error message")
			}
			return nil, errors.New(event.Error.ErrorMessage)
		}
	}
}

func writeError(writer http.ResponseWriter, status int, message string) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(map[string]any{
		"error": map[string]string{"message": message, "type": "server_error"},
	})
}
