package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {

	t.Setenv(
		"OPENAI_API_KEY",
		"test-key",
	)

	dir := t.TempDir()

	path := filepath.Join(
		dir,
		"config.yaml",
	)

	content := `
server:
  host: 0.0.0.0
  port: 8080

providers:
  openai:
    type: openai
    base_url: https://api.openai.com
    api_key: ${OPENAI_API_KEY}

models:
  gpt-5.6:
    provider: openai
    model: gpt-5.6
`

	if err := os.WriteFile(
		path,
		[]byte(content),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)

	if err != nil {
		t.Fatal(err)
	}

	if cfg.Server.Port != 8080 {
		t.Fatalf(
			"expected port 8080, got %d",
			cfg.Server.Port,
		)
	}

	if cfg.Providers["openai"].APIKey != "test-key" {
		t.Fatalf(
			"API key was not expanded",
		)
	}

	if cfg.Models["gpt-5.6"].Provider != "openai" {
		t.Fatalf(
			"model provider mismatch",
		)
	}
}
