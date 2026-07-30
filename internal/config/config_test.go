// 本文件验证配置未知字段拒绝和时间字段解析行为。
package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadRejectsUnknownFields 验证未知配置字段会被严格拒绝。
func TestLoadRejectsUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("server:\n  listen: ':8080'\n  typo: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("Load() error = nil, want unknown field error")
	}
}

// TestLoadParsesListenAddress 验证服务监听地址来自 YAML 配置。
func TestLoadParsesListenAddress(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("server:\n  listen: ':9090'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	config, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if config.Server.Listen != ":9090" {
		t.Fatalf("Listen = %q, want :9090", config.Server.Listen)
	}
}
