package config

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Host    string
	BaseURL string
}

func Parse() *Config {
	cfg := &Config{}

	defaultServerAddr := "localhost:8080"
	defaultBaseURL := "http://localhost:8080"

	envServerAddr := os.Getenv("SERVER_ADDRESS")
	envBaseURL := os.Getenv("BASE_URL")

	if envServerAddr != "" {
		cfg.Host = envServerAddr
	} else {
		cfg.Host = defaultServerAddr
	}

	if envBaseURL != "" {
		cfg.BaseURL = envBaseURL
	} else {
		cfg.BaseURL = defaultBaseURL
	}

	flagServerAddr := flag.String("a", "", "HTTP server address (overridden by SERVER_ADDRESS env)")
	flag.StringVar(flagServerAddr, "address", "", "HTTP server address (overridden by SERVER_ADDRESS env)")
	flagBaseURL := flag.String("b", "", "base address of shortened URL (overridden by BASE_URL env)")
	flag.StringVar(flagBaseURL, "base-url", "", "base address of shortened URL (overridden by BASE_URL env)")

	flag.Parse()

	if *flagServerAddr != "" && envServerAddr == "" {
		cfg.Host = *flagServerAddr
	}

	if *flagBaseURL != "" && envBaseURL == "" {
		cfg.BaseURL = *flagBaseURL
	}

	return cfg
}

func (c *Config) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("server address cannot be empty")
	}

	if !strings.HasPrefix(c.BaseURL, "http://") && !strings.HasPrefix(c.BaseURL, "https://") {
		return fmt.Errorf("base URL must start with http:// or https://")
	}

	c.BaseURL = strings.TrimSuffix(c.BaseURL, "/")

	return nil
}