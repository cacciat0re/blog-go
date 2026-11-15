package config

import (
	"fmt"

	"github.com/jinzhu/configor"
)

type Configs struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	CORS     CORSConfig     `yaml:"cors"`
}

type ServerConfig struct {
	Port int `yaml:"port" default:"8080"`
	// Timezone is the zone dates are displayed in, e.g. Asia/Shanghai. Empty
	// means UTC, so the host's clock never shifts what a reader sees.
	Timezone string `yaml:"timezone" default:"Asia/Shanghai"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host" default:"localhost"`
	Port     int    `yaml:"port" default:"5432"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Name     string `yaml:"name"`
}

type CORSConfig struct {
	AllowedOrigins []string `yaml:"allowed_origins"`
}

func Load(path string) (Configs, error) {
	var cfg Configs
	if err := configor.Load(&cfg, path); err != nil {
		return Configs{}, fmt.Errorf("load config %s: %w", path, err)
	}
	return cfg, nil
}
