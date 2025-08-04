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

type MoexPrices map[string]float64

type MoexQuery interface {
	FetchStocks(ctx context.Context) (MoexPrices, error)
	FetchBonds(ctx context.Context) (MoexPrices, error)
	FetchCurrencies(ctx context.Context) (MoexPrices, error)
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
func (mp moexPrices) getPriceList() MoexPrices {
	prices := make(MoexPrices)
	for _, position := range mp.Marketdata.Data {
		if len(position) != 2 {
			continue // Skip if the position does not have exactly two elements
		}
		secid, idIsOk := position[0].(string)
		price, priceIsOk := position[1].(float64)
		if idIsOk && priceIsOk {
			prices[secid] = price
		}
	}
	return prices
}

// ----------------------------------------------------------------
func (requester *MoexRequester) FetchStocks(ctx context.Context) (MoexPrices, error) {
	url := "https://iss.moex.com/iss/engines/stock/markets/shares/boards/TQBR/securities.json?iss.meta=off&iss.only=marketdata&marketdata.columns=SECID,LAST"
	prices, err := query[moexPrices](ctx, url)
	if err != nil {
		return nil, err
	}
	return prices.getPriceList(), nil
}

// ----------------------------------------------------------------
func (requester *MoexRequester) FetchBonds(ctx context.Context) (MoexPrices, error) {
	url := "https://iss.moex.com/iss/engines/stock/markets/bonds/boards/TQCB/securities.json?iss.meta=off&iss.only=marketdata&marketdata.columns=SECID,LAST"
	prices, err := query[moexPrices](ctx, url)
	if err != nil {
		return nil, err
	}
	return prices.getPriceList(), nil
}

// ----------------------------------------------------------------
func (requester *MoexRequester) FetchCurrencies(ctx context.Context) (MoexPrices, error) {
	url := "https://iss.moex.com/iss/engines/currency/markets/selt/boards/CETS/securities.json?iss.meta=off&iss.only=marketdata&marketdata.columns=SECID,LAST"
	prices, err := query[moexPrices](ctx, url)
	if err != nil {
		return nil, err
	}
	return prices.getPriceList(), nil
}

// ----------------------------------------------------------------
func newMoexRequester() MoexQuery {
	return &MoexRequester{}
}
