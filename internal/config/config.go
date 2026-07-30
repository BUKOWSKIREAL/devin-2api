// 本文件定义服务启动配置及其 YAML 加载和校验逻辑。
//
// Package config 负责加载和校验服务启动配置。
package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config 保存服务启动所需的全部配置；服务运行期间不会热更新。
type Config struct {
	// Server 保存 HTTP 服务配置。
	Server ServerConfig `yaml:"server"`
}

// ServerConfig 保存 HTTP 服务监听配置。
type ServerConfig struct {
	// Listen 是 HTTP 服务监听地址。
	Listen string `yaml:"listen"`
}

// Load 从 YAML 文件读取并校验配置。
func Load(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("open config %q: %w", path, err)
	}
	defer file.Close()

	var config Config
	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)
	if err := decoder.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("decode config %q: %w", path, err)
	}
	if err := config.Validate(); err != nil {
		return Config{}, fmt.Errorf("validate config %q: %w", path, err)
	}
	return config, nil
}

// Validate 检查配置中的必填项。
func (config Config) Validate() error {
	if config.Server.Listen == "" {
		return errors.New("server.listen is required")
	}
	return nil
}
