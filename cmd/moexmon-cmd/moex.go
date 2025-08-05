package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

type moexPrices struct {
	Marketdata struct {
		Columns []string `json:"columns"`
		Data    [][]any  `json:"data"`
	} `json:"marketdata"`
}

type MoexQuery interface {
	FetchPrice(ctx context.Context, asset string, assetType string) (float64, error)
}

type MoexRequester struct{}

// ----------------------------------------------------------------
func parseJSON[T any](s []byte) (T, error) {
	var r T
	if err := json.Unmarshal(s, &r); err != nil {
		slog.Error(fmt.Sprintf("failed to unmarshal JSON response: %s", err.Error()))
		return r, err
	}
	return r, nil
}

// ----------------------------------------------------------------
func query[T any](ctx context.Context, url string) (T, error) {
	var result T

	slog.Debug(fmt.Sprintf("Query MOEX: %s", url))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to create request: %s", err.Error()))
		return result, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to query MOEX: %s", err.Error()))
		return result, err
	}
	defer res.Body.Close() // nolint:errcheck,gosec

	body, err := io.ReadAll(res.Body)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to read response from MOEX: %s", err.Error()))
		return result, err
	}
	return parseJSON[T](body)
}

// ----------------------------------------------------------------
type AssetNotFoundError struct {
	Asset string
}

func (e *AssetNotFoundError) Error() string {
	return fmt.Sprintf("asset %s not found on MOEX", e.Asset)
}

// ----------------------------------------------------------------
func (requester *MoexRequester) FetchPrice(ctx context.Context, asset string, assetType string) (float64, error) {
	var market, mode string
	switch assetType {
	case "stock":
		market = "shares"
		mode = "TQBR"
	case "bond":
		market = "bonds"
		mode = "TQCB"
	case "currency":
		market = "currency"
		mode = "CETS"
	default:
		return 0, fmt.Errorf("unsupported asset type: %s", assetType)
	}

	url := fmt.Sprintf("https://iss.moex.com/iss/engines/stock/markets/%s/boards/%s/securities/%s.json?iss.meta=off&iss.only=marketdata&marketdata.columns=LAST",
		market, mode, asset)
	prices, err := query[moexPrices](ctx, url)
	if err != nil {
		return 0, err
	}

	if len(prices.Marketdata.Data) == 0 {
		return 0, &AssetNotFoundError{Asset: asset}
	}

	price, isOk := prices.Marketdata.Data[0][0].(float64)
	if !isOk {
		return 0, fmt.Errorf("invalid price data type for asset %s", asset)
	}

	return price, nil
}

// ----------------------------------------------------------------
func newMoexRequester() MoexQuery {
	return &MoexRequester{}
}
