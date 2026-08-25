package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port        int
	DataDir     string
	JWTSecret   string
	RateLimit   int
	LogLevel    string
	LogFormat   string
	CorsOrigins string
	TLSCert     string
	TLSKey      string
}

func Load() *Config {
	cfg := &Config{
		Port:        8080,
		DataDir:     "./data",
		RateLimit:   60,
		LogLevel:    "info",
		LogFormat:   "text",
		CorsOrigins: "",
	}
	if v := os.Getenv("PORT"); v != "" {
		cfg.Port, _ = strconv.Atoi(v)
	}
	if v := os.Getenv("DATA_DIR"); v != "" {
		cfg.DataDir = v
	}
	cfg.JWTSecret = os.Getenv("JWT_SECRET")
	if v := os.Getenv("RATE_LIMIT"); v != "" {
		cfg.RateLimit, _ = strconv.Atoi(v)
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}
	if v := os.Getenv("LOG_FORMAT"); v != "" {
		cfg.LogFormat = v
	}
	if v := os.Getenv("CORS_ORIGINS"); v != "" {
		cfg.CorsOrigins = v
	}
	cfg.TLSCert = os.Getenv("TLS_CERT")
	cfg.TLSKey = os.Getenv("TLS_KEY")
	return cfg
}

func (c *Config) DBPath() string {
	return c.DataDir + "/templates.db"
}

func (c *Config) BgImageDir() string {
	return c.DataDir + "/bg_images"
}

func (c *Config) OutputDir() string {
	return c.DataDir + "/output"
}

func (c *Config) FontsDir() string {
	return c.DataDir + "/fonts"
}
