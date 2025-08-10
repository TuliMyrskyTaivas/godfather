package godfather

import (
	"bytes"
	"os"
	"testing"
)

// ----------------------------------------------------------------
func TestSetupLogger_Verbose(t *testing.T) {
	// Capture output
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	logger := SetupLogger(true)
	logger.Debug("debug message")
	logger.Info("info message")

	w.Close() // nolint:all
	os.Stdout = oldStdout
	buf.ReadFrom(r) // nolint:all
	output := buf.String()

	if !contains(output, "debug message") {
		t.Errorf("Expected debug message in verbose mode, got: %s", output)
	}
	if !contains(output, "info message") {
		t.Errorf("Expected info message in verbose mode, got: %s", output)
	}
}

// ----------------------------------------------------------------
func TestSetupLogger_NonVerbose(t *testing.T) {
	// Capture output
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	logger := SetupLogger(false)
	logger.Debug("debug message")
	logger.Info("info message")

	w.Close() // nolint:all
	os.Stdout = oldStdout
	buf.ReadFrom(r) // nolint:all
	output := buf.String()

	if contains(output, "debug message") {
		t.Errorf("Did not expect debug message in non-verbose mode, got: %s", output)
	}
	if !contains(output, "info message") {
		t.Errorf("Expected info message in non-verbose mode, got: %s", output)
	}
}

// ----------------------------------------------------------------
func contains(s, substr string) bool {
	return len(s) > 0 && (s == substr || (len(s) > len(substr) && (s[0:len(substr)] == substr || contains(s[1:], substr))))
}
