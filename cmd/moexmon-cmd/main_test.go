package main

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/TuliMyrskyTaivas/godfather/internal/godfather"
)

// ----------------------------------------------------------------
func TestSetupLogger_Verbose(t *testing.T) {
	// Capture output
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	logger := setupLogger(true)
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

	logger := setupLogger(false)
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
type mockMoexQuery struct {
	price float64
	err   error
}

func (m *mockMoexQuery) FetchPrice(ctx context.Context, ticker string, assetClass string) (float64, error) {
	return m.price, m.err
}

// ----------------------------------------------------------------
func TestConditionMatch_AboveConditionMet(t *testing.T) {
	item := godfather.MOEXWatchlistItem{
		Ticker:      "AAPL",
		AssetClass:  "stock",
		Condition:   "above",
		TargetPrice: 100.0,
	}
	moex := &mockMoexQuery{price: 150.0}
	result := conditionMatch(context.Background(), item, moex)
	if !result {
		t.Errorf("Expected true for price above target")
	}
}

// ----------------------------------------------------------------
func TestConditionMatch_AboveConditionNotMet(t *testing.T) {
	item := godfather.MOEXWatchlistItem{
		Ticker:      "AAPL",
		AssetClass:  "stock",
		Condition:   "above",
		TargetPrice: 200.0,
	}
	moex := &mockMoexQuery{price: 150.0}
	result := conditionMatch(context.Background(), item, moex)
	if result {
		t.Errorf("Expected false for price not above target")
	}
}

// ----------------------------------------------------------------
func TestConditionMatch_BelowConditionMet(t *testing.T) {
	item := godfather.MOEXWatchlistItem{
		Ticker:      "AAPL",
		AssetClass:  "stock",
		Condition:   "below",
		TargetPrice: 200.0,
	}
	moex := &mockMoexQuery{price: 150.0}
	result := conditionMatch(context.Background(), item, moex)
	if !result {
		t.Errorf("Expected true for price below target")
	}
}

// ----------------------------------------------------------------
func TestConditionMatch_BelowConditionNotMet(t *testing.T) {
	item := godfather.MOEXWatchlistItem{
		Ticker:      "AAPL",
		AssetClass:  "stock",
		Condition:   "below",
		TargetPrice: 100.0,
	}
	moex := &mockMoexQuery{price: 150.0}
	result := conditionMatch(context.Background(), item, moex)
	if result {
		t.Errorf("Expected false for price not below target")
	}
}

// ----------------------------------------------------------------
func TestConditionMatch_UnknownCondition(t *testing.T) {
	item := godfather.MOEXWatchlistItem{
		Ticker:      "AAPL",
		AssetClass:  "stock",
		Condition:   "unknown",
		TargetPrice: 100.0,
	}
	moex := &mockMoexQuery{price: 150.0}
	result := conditionMatch(context.Background(), item, moex)
	if result {
		t.Errorf("Expected false for unknown condition")
	}
}

// ----------------------------------------------------------------
func TestConditionMatch_AssetNotFoundError(t *testing.T) {
	item := godfather.MOEXWatchlistItem{
		Ticker:      "AAPL",
		AssetClass:  "stock",
		Condition:   "above",
		TargetPrice: 100.0,
	}
	moex := &mockMoexQuery{err: &AssetNotFoundError{}}
	result := conditionMatch(context.Background(), item, moex)
	if result {
		t.Errorf("Expected false when AssetNotFoundError is returned")
	}
}

// ----------------------------------------------------------------
func TestConditionMatch_OtherError(t *testing.T) {
	item := godfather.MOEXWatchlistItem{
		Ticker:      "AAPL",
		AssetClass:  "stock",
		Condition:   "above",
		TargetPrice: 100.0,
	}
	moex := &mockMoexQuery{err: os.ErrInvalid}
	result := conditionMatch(context.Background(), item, moex)
	if result {
		t.Errorf("Expected false when other error is returned")
	}
}

// ----------------------------------------------------------------
func contains(s, substr string) bool {
	return len(s) > 0 && (s == substr || (len(s) > len(substr) && (s[0:len(substr)] == substr || contains(s[1:], substr))))
}
