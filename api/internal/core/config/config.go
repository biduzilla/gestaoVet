package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Port int `yaml:"port"`
	DB   struct {
		DSN          string `yaml:"dsn"`
		MaxOpenConns int    `yaml:"max_open_conns"`
		MaxIdleConns int    `yaml:"max_idle_conns"`
		MaxIdleTime  string `yaml:"max_idle_time"`
	} `yaml:"db"`
	Limiter struct {
		RPS     float64 `yaml:"rps"`
		Burst   int     `yaml:"burst"`
		Enabled bool    `yaml:"enabled"`
	} `yaml:"limiter"`
	CORS struct {
		TrustedOrigins []string `yaml:"trusted_origins"`
	} `yaml:"cors"`
	Security struct {
		SecretKey string `yaml:"secret_key"`
	} `yaml:"security"`
}

func Load() (*Config, error) {
	data, err := os.ReadFile("resource/config.yml")
	if err != nil {
		return nil, err
	}

	var config Config

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, err
}
