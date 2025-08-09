package main

import (
	"os"
	"path/filepath"
	"testing"
)

// ----------------------------------------------------------------
func TestParseConfig_ValidFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "config-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name()) //nolint:errcheck

	configContent := `{
		"check_interval_seconds": 10,
		"prometheus": {
			"port": 9090,
			"url": "http://localhost:9090"
		},
		"database": {
			"host": "localhost",
			"port": 5432,
			"user": "testuser",
			"passwd": "testpass",
			"database": "testdb"
		},
		"nats": {
			"host": "localhost",
			"port": 4222
		}
	}`
	if _, err := tmpFile.Write([]byte(configContent)); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tmpFile.Close() //nolint:gosec,errcheck

	cfg, err := ParseConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("ParseConfig failed: %v", err)
	}
	if cfg.CheckIntervalSeconds != 10 {
		t.Errorf("expected CheckIntervalSeconds=10, got %d", cfg.CheckIntervalSeconds)
	}
	if cfg.Prometheus.Port != 9090 || cfg.Prometheus.URL != "http://localhost:9090" {
		t.Errorf("unexpected Prometheus config: %+v", cfg.Prometheus)
	}
	if cfg.Database.Host != "localhost" || cfg.Database.Port != 5432 ||
		cfg.Database.User != "testuser" || cfg.Database.Passwd != "testpass" ||
		cfg.Database.Database != "testdb" {
		t.Errorf("unexpected Database config: %+v", cfg.Database)
	}
	if cfg.NATS.Host != "localhost" || cfg.NATS.Port != 4222 {
		t.Errorf("unexpected NATS config: %+v", cfg.NATS)
	}
}

// ----------------------------------------------------------------
func TestParseConfig_FileNotFound(t *testing.T) {
	_, err := ParseConfig("nonexistent-file.json")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

// ----------------------------------------------------------------
func TestParseConfig_InvalidJSON(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "config-invalid-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name()) //nolint:errcheck

	invalidContent := `{"check_interval_seconds": 10,`
	if _, err := tmpFile.Write([]byte(invalidContent)); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tmpFile.Close() //nolint:gosec,errcheck

	_, err = ParseConfig(tmpFile.Name())
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

// ----------------------------------------------------------------
func TestParseConfig_PathSanitization(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "config-path-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name()) //nolint:errcheck

	configContent := `{"check_interval_seconds": 5, "prometheus": {"port": 1, "url": ""}, "database": {"host": "", "port": 0, "user": "", "passwd": "", "database": ""}}`
	if _, err := tmpFile.Write([]byte(configContent)); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tmpFile.Close() //nolint:gosec,errcheck

	// Use a path with redundant elements
	path := filepath.Join(filepath.Dir(tmpFile.Name()), ".", filepath.Base(tmpFile.Name()))
	cfg, err := ParseConfig(path)
	if err != nil {
		t.Fatalf("ParseConfig failed: %v", err)
	}
	if cfg.CheckIntervalSeconds != 5 {
		t.Errorf("expected CheckIntervalSeconds=5, got %d", cfg.CheckIntervalSeconds)
	}
}
