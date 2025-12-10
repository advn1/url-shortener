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
	FileStoragePath string
}

func Parse() *Config {
	cfg := &Config{}

	defaultServerAddr := "localhost:8080"
	defaultBaseURL := "http://localhost:8080"
	defaultStoragePath := "/tmp/short-url-db.json"

	envServerAddr := os.Getenv("SERVER_ADDRESS")
	envBaseURL := os.Getenv("BASE_URL")
	envFileStoragePath := os.Getenv("FILE_STORAGE_PATH")

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

	if envFileStoragePath != "" {
		cfg.FileStoragePath = envFileStoragePath
	} else {
		cfg.FileStoragePath = defaultStoragePath
	}

	flagServerAddr := flag.String("a", "", "HTTP server address (overridden by SERVER_ADDRESS env)")
	flag.StringVar(flagServerAddr, "address", "", "HTTP server address (overridden by SERVER_ADDRESS env)")
	flagBaseURL := flag.String("b", "", "base address of shortened URL (overridden by BASE_URL env)")
	flag.StringVar(flagBaseURL, "base-url", "", "base address of shortened URL (overridden by BASE_URL env)")
	flagFileStoragePath := flag.String("f", "", "path of storage file of shortened URLs (overridden by FILE_STORAGE_PATH env)")
	flag.StringVar(flagFileStoragePath, "file", "", "path of storage file of shortened URLs (overridden by FILE_STORAGE_PATH env)")
	fmt.Println(*flagFileStoragePath)
	
	flag.Parse()

	if *flagServerAddr != "" && envServerAddr == "" {
		cfg.Host = *flagServerAddr
	}

	if *flagBaseURL != "" && envBaseURL == "" {
		cfg.BaseURL = *flagBaseURL
	}

	if *flagFileStoragePath != "" && envFileStoragePath == "" {
		cfg.FileStoragePath = *flagFileStoragePath
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

	// here add checking for file storage path

	c.BaseURL = strings.TrimSuffix(c.BaseURL, "/")

	return nil
}