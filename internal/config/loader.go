package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

func Load(path string) (*Config, error) {

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf(
			"read config file: %w",
			err,
		)
	}

	content := os.ExpandEnv(
		string(data),
	)

	var cfg Config

	if err := yaml.Unmarshal(
		[]byte(content),
		&cfg,
	); err != nil {
		return nil, fmt.Errorf(
			"parse config file: %w",
			err,
		)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Validate() error {

	if c.Server.Port <= 0 {
		return fmt.Errorf(
			"server.port must be greater than 0",
		)
	}

	if len(c.Providers) == 0 {
		return fmt.Errorf(
			"no providers configured",
		)
	}

	if len(c.Models) == 0 {
		return fmt.Errorf(
			"no models configured",
		)
	}

	for name, p := range c.Providers {

		if strings.TrimSpace(p.Type) == "" {
			return fmt.Errorf(
				"provider '%s': type is required",
				name,
			)
		}
	}

	for name, model := range c.Models {

		if strings.TrimSpace(model.Provider) == "" {
			return fmt.Errorf(
				"model '%s': provider is required",
				name,
			)
		}

		if _, ok := c.Providers[model.Provider]; !ok {
			return fmt.Errorf(
				"model '%s': provider '%s' not found",
				name,
				model.Provider,
			)
		}
	}

	return nil
}
