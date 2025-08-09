package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ----------------------------------------------------------------
type Config struct {
	CheckIntervalSeconds int `json:"check_interval_seconds"`
	Prometheus           struct {
		Port int    `json:"port"`
		URL  string `json:"url"`
	} `json:"prometheus"`
	Database struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		User     string `json:"user"`
		Passwd   string `json:"passwd"`
		Database string `json:"database"`
	} `json:"database"`
	NATS struct {
		Host string `json:"host"`
		Port int    `json:"port"`
		User string `json:"user"`
		Pass string `json:"pass"`
	} `json:"nats"`
}

// ----------------------------------------------------------------
func ParseConfig(path string) (*Config, error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck

	var cfg Config
	decoder := json.NewDecoder(f)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
