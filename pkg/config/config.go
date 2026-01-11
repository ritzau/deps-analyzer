package config

import (
	"fmt"
	"strings"

	"github.com/knadh/koanf/parsers/toml/v2"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	"github.com/spf13/pflag"
)

// Config holds all configuration for the application
type Config struct {
	Workspace   string `koanf:"workspace"`
	WebMode     bool   `koanf:"web"`
	Port        int    `koanf:"port"`
	Watch       bool   `koanf:"watch"`
	OpenBrowser bool   `koanf:"open"`
	Licenses    bool   `koanf:"licenses"`
	Verbosity   string `koanf:"verbosity"`
	VerboseCnt  int    `koanf:"verbose"`
}

// Load loads configuration from defaults, config file, environment variables, and flags.
// Priority: Flags > Env > Config File > Defaults
func Load(f *pflag.FlagSet) (*Config, error) {
	k := koanf.New(".")

	// 1. Defaults
	defaults := map[string]interface{}{
		"workspace": ".",
		"web":       false,
		"port":      8080,
		"watch":     false,
		"open":      true,
		"licenses":  false,
		"verbosity": "",
		"verbose":   0,
	}
	if err := k.Load(makeMapProvider(defaults), nil); err != nil {
		return nil, fmt.Errorf("failed to load defaults: %w", err)
	}

	// 2. Config File (optional) - deps-analyzer.toml
	// We ignore errors here as the file might not exist
	_ = k.Load(file.Provider("deps-analyzer.toml"), toml.Parser())

	// 3. Environment Variables
	// Prefix: DEPS_ANALYZER_ (e.g., DEPS_ANALYZER_PORT=9090)
	if err := k.Load(env.Provider("DEPS_ANALYZER_", ".", func(s string) string {
		return strings.ReplaceAll(strings.ToLower(
			strings.TrimPrefix(s, "DEPS_ANALYZER_")), "_", ".")
	}), nil); err != nil {
		return nil, fmt.Errorf("failed to load env vars: %w", err)
	}

	// 4. Flags
	if f != nil {
		if err := k.Load(posflag.Provider(f, ".", k), nil); err != nil {
			return nil, fmt.Errorf("failed to load flags: %w", err)
		}
	}

	// Unmarshal into struct
	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// Helper to use map as a provider
type mapProvider struct {
	m map[string]interface{}
}

func makeMapProvider(m map[string]interface{}) *mapProvider {
	return &mapProvider{m: m}
}

func (p *mapProvider) Read() (map[string]interface{}, error) {
	return p.m, nil
}

func (p *mapProvider) ReadBytes() ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}
