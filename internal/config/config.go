package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	Airport string    `json:"airport"`
	Format  string    `json:"format"`
	Modules ModuleCfg `json:"modules"`
}

type ModuleCfg struct {
	METAR  bool `json:"metar"`
	TAF    bool `json:"taf"`
	AFD    bool `json:"discussion"`
	AIRMET bool `json:"airmet"`
	PIREP  bool `json:"pirep"`
}

const defaultFormat = "{temps} {vis} {cloud-icon} {clouds} {wx}"

func Load() *Config {
	defaults := &Config{
		Format:  defaultFormat,
		Modules: ModuleCfg{METAR: true},
	}

	path, err := configPath()
	if err != nil {
		return defaults
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return defaults
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return defaults
	}

	if cfg.Format == "" {
		cfg.Format = defaults.Format
	}

	return &cfg
}

func configPath() (string, error) {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "pilot-bar", "config.json"), nil
}
