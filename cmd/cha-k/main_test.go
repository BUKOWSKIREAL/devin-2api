// 本文件验证服务入口能向调用方返回服务器启动失败。
package main

import (
	"context"
	"errors"
	"testing"
)

type fakeServer struct {
	shutdown bool
}

func (server *fakeServer) ListenAndServe() error {
	return errors.New("listen stopped")
}

func (server *fakeServer) Shutdown(context.Context) error {
	server.shutdown = true
	return nil
}

// TestRunReturnsServeError verifies unexpected server failures are returned to main.
func TestRunReturnsServeError(t *testing.T) {
	server := &fakeServer{}
	if err := run(context.Background(), server); err == nil {
		t.Fatal("run() error = nil, want serve error")
	}
	if server.shutdown {
		t.Fatal("server was shut down after an unexpected serve failure")
	}
}
