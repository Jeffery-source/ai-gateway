package config

import "fmt"

type ProviderConfig struct {
	Type    string `yaml:"type"`
	BaseURL string `yaml:"base_url"`
	APIKey  string `yaml:"api_key"`
}

type ModelConfig struct {
	Provider string `yaml:"provider"`
	Model    string `yaml:"model"`
}

type Config struct {
	Server    ServerConfig              `yaml:"server"`
	Providers map[string]ProviderConfig `yaml:"providers"`
	Models    map[string]ModelConfig    `yaml:"models"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

func (c ServerConfig) Address() string {
	return fmt.Sprintf(
		"%s:%d",
		c.Host,
		c.Port,
	)
}
