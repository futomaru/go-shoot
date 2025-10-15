package config

import (
	"fmt"
	"os"
	"strings"
)

// Config holds runtime configuration for the Go Shoot backend services.
type Config struct {
	HTTPPort       string
	AllowedOrigins []string
}

// Load reads environment variables and returns a populated Config.
func Load() Config {
	port := os.Getenv("GO_SHOOT_HTTP_PORT")
	if port == "" {
		port = "8080"
	}

	origins := os.Getenv("GO_SHOOT_ALLOWED_ORIGINS")
	allowedOrigins := []string{"*"}
	if origins != "" {
		allowedOrigins = splitAndTrim(origins)
	}

	return Config{
		HTTPPort:       port,
		AllowedOrigins: allowedOrigins,
	}
}

func (c Config) Address() string {
	return fmt.Sprintf(":%s", c.HTTPPort)
}

func splitAndTrim(input string) []string {
	raw := strings.Split(input, ",")
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return []string{"*"}
	}
	return out
}
