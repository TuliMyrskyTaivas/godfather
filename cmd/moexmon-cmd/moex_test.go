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
func TestFetchPrice_StockSuccess(t *testing.T) {
	// Mock HTTP response for stock
	body := `{"marketdata":{"columns":["LAST"],"data":[[123.45]]}}`
	mockResp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
	client := &http.Client{Transport: &mockRoundTripper{resp: mockResp}}
	http.DefaultClient = client

	requester := &MoexRequester{}
	ctx := context.Background()
	price, err := requester.FetchPrice(ctx, "AAPL", "stock")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if price != 123.45 {
		t.Errorf("expected price 123.45, got %v", price)
	}
}

// ----------------------------------------------------------------
func TestFetchPrice_BondSuccess(t *testing.T) {
	// Mock HTTP response for bond
	body := `{"marketdata":{"columns":["LAST"],"data":[[101.01]]}}`
	mockResp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
	client := &http.Client{Transport: &mockRoundTripper{resp: mockResp}}
	http.DefaultClient = client

	requester := &MoexRequester{}
	ctx := context.Background()
	price, err := requester.FetchPrice(ctx, "RU000A0JX0J2", "bond")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if price != 101.01 {
		t.Errorf("expected price 101.01, got %v", price)
	}
}

// ----------------------------------------------------------------
func TestFetchPrice_CurrencySuccess(t *testing.T) {
	// Mock HTTP response for currency
	body := `{"marketdata":{"columns":["LAST"],"data":[[74.32]]}}`
	mockResp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
	client := &http.Client{Transport: &mockRoundTripper{resp: mockResp}}
	http.DefaultClient = client

	requester := &MoexRequester{}
	ctx := context.Background()
	price, err := requester.FetchPrice(ctx, "USD_RUB", "currency")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if price != 74.32 {
		t.Errorf("expected price 74.32, got %v", price)
	}
}

// ----------------------------------------------------------------
func TestFetchPrice_UnsupportedAssetType(t *testing.T) {
	requester := &MoexRequester{}
	ctx := context.Background()
	_, err := requester.FetchPrice(ctx, "AAPL", "crypto")
	if err == nil {
		t.Error("expected error for unsupported asset type, got nil")
	}
}

// ----------------------------------------------------------------
func TestFetchPrice_NoPriceData(t *testing.T) {
	// Mock HTTP response with no data
	body := `{"marketdata":{"columns":["LAST"],"data":[]}}`
	mockResp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
	client := &http.Client{Transport: &mockRoundTripper{resp: mockResp}}
	http.DefaultClient = client

	requester := &MoexRequester{}
	ctx := context.Background()
	_, err := requester.FetchPrice(ctx, "AAPL", "stock")
	if err == nil || err.Error() != "asset AAPL not found on MOEX" {
		t.Errorf("expected 'asset AAPL not found on MOEX' error, got %v", err)
	}
}

// ----------------------------------------------------------------
func TestFetchPrice_InvalidPriceType(t *testing.T) {
	// Mock HTTP response with wrong type
	body := `{"marketdata":{"columns":["LAST"],"data":[["not_a_float"]]}}`
	mockResp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
	client := &http.Client{Transport: &mockRoundTripper{resp: mockResp}}
	http.DefaultClient = client

	requester := &MoexRequester{}
	ctx := context.Background()
	_, err := requester.FetchPrice(ctx, "AAPL", "stock")
	if err == nil || err.Error() == "" {
		t.Error("expected error for invalid price data type, got nil")
	}
}

// ----------------------------------------------------------------
func TestFetchPrice_QueryError(t *testing.T) {
	// Simulate HTTP client error
	client := &http.Client{Transport: &mockRoundTripper{resp: nil, err: errors.New("network error")}}
	http.DefaultClient = client

	requester := &MoexRequester{}
	ctx := context.Background()
	_, err := requester.FetchPrice(ctx, "AAPL", "stock")
	if err == nil {
		t.Error("expected error from query, got nil")
	}
}
