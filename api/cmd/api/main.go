package main

import (
	"gestaoVet/internal/api"
	"gestaoVet/internal/core/config"
)

func main() {
	c := config.New()
	var cfg config.Config

	cfg.Server.Port = c.Server.Port
	cfg.Server.Timeout = c.Server.Timeout
	cfg.Env = "development"
	cfg.DB.DSN = c.DB.DSN
	cfg.DB.MaxOpenConns = c.DB.MaxOpenConns
	cfg.DB.MaxIdleConns = c.DB.MaxIdleConns
	cfg.DB.MaxIdleTime = c.DB.MaxIdleTime
	cfg.Limiter.RPS = c.RateLimiter.RPS
	cfg.Limiter.Burst = c.RateLimiter.Burst
	cfg.Limiter.Enabled = c.RateLimiter.Enabled
	cfg.Security.PrivateKeyPath = c.Security.PrivateKeyPath
	cfg.Security.PublicKeyPath = c.Security.PublicKeyPath

	app := api.NewApp(cfg)
	err := app.Server()
	if err != nil {
		app.Logger.PrintFatal(err, nil)
	}
}
