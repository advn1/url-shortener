package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	ServerAddr    string
	BaseURL string
	FileStoragePath string
	DatabaseDSN string
}

func setValue(envValue string, flagValue string, defaultValue string) string {
	if envValue != "" {
		return strings.TrimSpace(envValue)
	}
	if flagValue != "" {
		return strings.TrimSpace(flagValue) 
	}
	return strings.TrimSpace(defaultValue)
}

func Parse() *Config {
	cfg := &Config{
		ServerAddr:      "localhost:8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "",
		DatabaseDSN:     "", // host=localhost user=postgres password=1234 dbname=postgres sslmode=disable
	}

	envServerAddr := strings.TrimSpace(os.Getenv("SERVER_ADDRESS"))
	envBaseURL := strings.TrimSpace(os.Getenv("BASE_URL"))
	envFileStoragePath := strings.TrimSpace(os.Getenv("FILE_STORAGE_PATH"))
	envDatabaseDSN := strings.TrimSpace(os.Getenv("DATABASE_DSN"))
	
	flagServerAddr := flag.String("a", "", "HTTP server address (overridden by SERVER_ADDRESS env)")
	flag.StringVar(flagServerAddr, "address", "", "HTTP server address (overridden by SERVER_ADDRESS env)")
	flagBaseURL := flag.String("b", "", "base address of shortened URL (overridden by BASE_URL env)")
	flag.StringVar(flagBaseURL, "base-url", "", "base address of shortened URL (overridden by BASE_URL env)")
	flagFileStoragePath := flag.String("f", "", "path of storage file of shortened URLs (overridden by FILE_STORAGE_PATH env)")
	flag.StringVar(flagFileStoragePath, "file", "", "path of storage file of shortened URLs (overridden by FILE_STORAGE_PATH env)")
	flagDatabaseDSN := flag.String("d", "", "database dsn (data source name). stores all connection details (overridden by DATABASE_DSN env)")
	flag.StringVar(flagDatabaseDSN, "database", "", "database dsn (data source name). stores all connection details (overridden by DATABASE_DSN env)")
	
	flag.Parse()

	cfg.ServerAddr = setValue(envServerAddr, *flagServerAddr, cfg.ServerAddr)
	cfg.BaseURL = setValue(envBaseURL, *flagBaseURL, cfg.BaseURL)
	cfg.FileStoragePath = setValue(envFileStoragePath, *flagFileStoragePath, cfg.FileStoragePath)
	cfg.DatabaseDSN = setValue(envDatabaseDSN, *flagDatabaseDSN, cfg.DatabaseDSN)

	return cfg
}

func (c *Config) Validate() error {
	errs := make([]error, 0, 3)

	if c.ServerAddr == "" {
		errs = append(errs, fmt.Errorf("server address cannot be empty"))
	}

	if !strings.HasPrefix(c.BaseURL, "http://") && !strings.HasPrefix(c.BaseURL, "https://") {
		errs = append(errs,fmt.Errorf("base URL must start with http:// or https://"))
	}

	return errors.Join(errs...)
}