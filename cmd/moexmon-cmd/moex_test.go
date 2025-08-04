package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
)

// ----------------------------------------------------------------
func TestParseJSON_Success(t *testing.T) {
	jsonStr := `{"marketdata":{"columns":["SECID","LAST"],"data":[["AAPL",123.45],["GOOG",234.56]]}}`
	result, err := parseJSON[moexPrices]([]byte(jsonStr))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Marketdata.Data) != 2 {
		t.Errorf("expected 2 data entries, got %d", len(result.Marketdata.Data))
	}
	if result.Marketdata.Columns[0] != "SECID" || result.Marketdata.Columns[1] != "LAST" {
		t.Errorf("unexpected columns: %v", result.Marketdata.Columns)
	}
}

// ----------------------------------------------------------------
func TestParseJSON_InvalidJSON(t *testing.T) {
	invalidJSON := []byte(`{invalid json}`)
	_, err := parseJSON[moexPrices](invalidJSON)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

// ----------------------------------------------------------------
func TestParseJSON_EmptyJSON(t *testing.T) {
	emptyJSON := []byte(`{}`)
	result, err := parseJSON[moexPrices](emptyJSON)
	if err != nil {
		t.Fatalf("expected no error for empty JSON, got %v", err)
	}
	// Marketdata should be zero value
	if len(result.Marketdata.Columns) != 0 || len(result.Marketdata.Data) != 0 {
		t.Errorf("expected empty columns and data, got %v %v", result.Marketdata.Columns, result.Marketdata.Data)
	}
}

// ----------------------------------------------------------------
// mockRoundTripper implements http.RoundTripper for testing
type mockRoundTripper struct {
	resp *http.Response
	err  error
}

// ----------------------------------------------------------------
func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.resp, m.err
}

// ----------------------------------------------------------------
func TestQuery_Success(t *testing.T) {
	// Prepare mock response
	body := `{"marketdata":{"columns":["SECID","LAST"],"data":[["AAPL",123.45]]}}`
	mockResp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
	client := &http.Client{Transport: &mockRoundTripper{resp: mockResp}}
	http.DefaultClient = client

	ctx := context.Background()
	result, err := query[moexPrices](ctx, "http://example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Marketdata.Data) != 1 {
		t.Errorf("expected 1 data entry, got %d", len(result.Marketdata.Data))
	}
}

// ----------------------------------------------------------------
func TestQuery_RequestError(t *testing.T) {
	// Simulate error in request creation by passing invalid URL
	ctx := context.Background()
	_, err := query[moexPrices](ctx, "http://[::1]:namedport")
	if err == nil {
		t.Error("expected error for invalid URL, got nil")
	}
}

// ----------------------------------------------------------------
func TestQuery_HTTPError(t *testing.T) {
	// Simulate HTTP client error
	client := &http.Client{Transport: &mockRoundTripper{resp: nil, err: errors.New("network error")}}
	http.DefaultClient = client

	ctx := context.Background()
	_, err := query[moexPrices](ctx, "http://example.com")
	if err == nil {
		t.Error("expected error for HTTP client, got nil")
	}
}

// ----------------------------------------------------------------
func TestQuery_ReadError(t *testing.T) {
	// Simulate error reading response body
	r := io.NopCloser(badReader{})
	mockResp := &http.Response{
		StatusCode: 200,
		Body:       r,
	}
	client := &http.Client{Transport: &mockRoundTripper{resp: mockResp}}
	http.DefaultClient = client

	ctx := context.Background()
	_, err := query[moexPrices](ctx, "http://example.com")
	if err == nil {
		t.Error("expected error reading body, got nil")
	}
}

type badReader struct{}

func (badReader) Read(p []byte) (int, error) { return 0, errors.New("read error") }
func (badReader) Close() error               { return nil }

// ----------------------------------------------------------------
func TestGetPriceList_ValidData(t *testing.T) {
	mp := moexPrices{
		Marketdata: struct {
			Columns []string `json:"columns"`
			Data    [][]any  `json:"data"`
		}{
			Columns: []string{"SECID", "LAST"},
			Data: [][]any{
				{"AAPL", 123.45},
				{"GOOG", 234.56},
			},
		},
	}
	prices := mp.getPriceList()
	if len(prices) != 2 {
		t.Errorf("expected 2 prices, got %d", len(prices))
	}
	if prices["AAPL"] != 123.45 {
		t.Errorf("expected AAPL price 123.45, got %v", prices["AAPL"])
	}
	if prices["GOOG"] != 234.56 {
		t.Errorf("expected GOOG price 234.56, got %v", prices["GOOG"])
	}
}

// ----------------------------------------------------------------
func TestGetPriceList_InvalidTypes(t *testing.T) {
	mp := moexPrices{
		Marketdata: struct {
			Columns []string `json:"columns"`
			Data    [][]any  `json:"data"`
		}{
			Columns: []string{"SECID", "LAST"},
			Data: [][]any{
				{123, 123.45},        // SECID not string
				{"AAPL", "notfloat"}, // price not float64
				{"GOOG", 234.56},     // valid
			},
		},
	}
	prices := mp.getPriceList()
	if len(prices) != 1 {
		t.Errorf("expected 1 valid price, got %d", len(prices))
	}
	if prices["GOOG"] != 234.56 {
		t.Errorf("expected GOOG price 234.56, got %v", prices["GOOG"])
	}
}

// ----------------------------------------------------------------
func TestGetPriceList_EmptyData(t *testing.T) {
	mp := moexPrices{
		Marketdata: struct {
			Columns []string `json:"columns"`
			Data    [][]any  `json:"data"`
		}{
			Columns: []string{"SECID", "LAST"},
			Data:    [][]any{},
		},
	}
	prices := mp.getPriceList()
	if len(prices) != 0 {
		t.Errorf("expected 0 prices, got %d", len(prices))
	}
}

// ----------------------------------------------------------------
func TestGetPriceList_WrongLengthData(t *testing.T) {
	mp := moexPrices{
		Marketdata: struct {
			Columns []string `json:"columns"`
			Data    [][]any  `json:"data"`
		}{
			Columns: []string{"SECID", "LAST"},
			Data: [][]any{
				{"AAPL"},                // only one element
				{"GOOG", 234.56, "foo"}, // three elements
				{"MSFT", 345.67},        // valid
			},
		},
	}
	prices := mp.getPriceList()
	if len(prices) != 1 {
		t.Errorf("expected 1 valid price, got %d", len(prices))
	}
	if prices["MSFT"] != 345.67 {
		t.Errorf("expected MSFT price 345.67, got %v", prices["MSFT"])
	}
}

// ----------------------------------------------------------------
func TestFetchStocks_Success(t *testing.T) {
	// Prepare mock response
	body := `{"marketdata":{"columns":["SECID","LAST"],"data":[["AAPL",123.45],["GOOG",234.56]]}}`
	mockResp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
	client := &http.Client{Transport: &mockRoundTripper{resp: mockResp}}
	http.DefaultClient = client

	ctx := context.Background()
	requester := &MoexRequester{}
	prices, err := requester.FetchStocks(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(prices) != 2 {
		t.Errorf("expected 2 prices, got %d", len(prices))
	}
	if prices["AAPL"] != 123.45 {
		t.Errorf("expected AAPL price 123.45, got %v", prices["AAPL"])
	}
	if prices["GOOG"] != 234.56 {
		t.Errorf("expected GOOG price 234.56, got %v", prices["GOOG"])
	}
}

// ----------------------------------------------------------------
func TestFetchStocks_QueryError(t *testing.T) {
	// Simulate HTTP client error
	client := &http.Client{Transport: &mockRoundTripper{resp: nil, err: errors.New("network error")}}
	http.DefaultClient = client

	ctx := context.Background()
	requester := &MoexRequester{}
	_, err := requester.FetchStocks(ctx)
	if err == nil {
		t.Error("expected error from FetchStocks, got nil")
	}
}

// ----------------------------------------------------------------
func TestFetchStocks_InvalidJSON(t *testing.T) {
	// Prepare mock response with invalid JSON
	body := `{invalid json}`
	mockResp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
	client := &http.Client{Transport: &mockRoundTripper{resp: mockResp}}
	http.DefaultClient = client

	ctx := context.Background()
	requester := &MoexRequester{}
	_, err := requester.FetchStocks(ctx)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

// ----------------------------------------------------------------
func TestFetchStocks_EmptyData(t *testing.T) {
	// Prepare mock response with empty data
	body := `{"marketdata":{"columns":["SECID","LAST"],"data":[]}}`
	mockResp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
	client := &http.Client{Transport: &mockRoundTripper{resp: mockResp}}
	http.DefaultClient = client

	ctx := context.Background()
	requester := &MoexRequester{}
	prices, err := requester.FetchStocks(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(prices) != 0 {
		t.Errorf("expected 0 prices, got %d", len(prices))
	}
}
