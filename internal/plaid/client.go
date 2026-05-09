package plaid

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL  string
	clientID string
	secret   string
	http     *http.Client
}

func NewClient(baseURL, clientID, secret string) *Client {
	return &Client{
		baseURL:  strings.TrimRight(baseURL, "/"),
		clientID: clientID,
		secret:   secret,
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type LinkTokenResponse struct {
	LinkToken string `json:"link_token"`
	Expiration string `json:"expiration"`
	RequestID string `json:"request_id"`
}

type ExchangePublicTokenResponse struct {
	AccessToken string `json:"access_token"`
	ItemID      string `json:"item_id"`
	RequestID   string `json:"request_id"`
}

type Account struct {
	AccountID    string `json:"account_id"`
	Name         string `json:"name"`
	OfficialName string `json:"official_name"`
	Type         string `json:"type"`
	Subtype      string `json:"subtype"`
	Mask         string `json:"mask"`
}

type Transaction struct {
	AccountID       string  `json:"account_id"`
	TransactionID   string  `json:"transaction_id"`
	Date            string  `json:"date"`
	Name            string  `json:"name"`
	MerchantName    string  `json:"merchant_name"`
	Amount          float64 `json:"amount"`
	ISOCurrencyCode string  `json:"iso_currency_code"`
	Pending         bool    `json:"pending"`
}

type RemovedTransaction struct {
	TransactionID string `json:"transaction_id"`
	AccountID     string `json:"account_id"`
}

type TransactionsSyncResponse struct {
	Accounts   []Account            `json:"accounts"`
	Added      []Transaction        `json:"added"`
	Modified   []Transaction        `json:"modified"`
	Removed    []RemovedTransaction `json:"removed"`
	NextCursor string               `json:"next_cursor"`
	HasMore    bool                 `json:"has_more"`
	RequestID  string               `json:"request_id"`
}

func (c *Client) CreateLinkToken(ctx context.Context, userID, webhookURL string) (LinkTokenResponse, error) {
	req := map[string]any{
		"client_id":    c.clientID,
		"secret":       c.secret,
		"client_name":  "NetFlow Sandbox",
		"country_codes": []string{"US"},
		"language":     "en",
		"products":     []string{"transactions"},
		"user": map[string]string{
			"client_user_id": userID,
		},
	}
	if webhookURL != "" {
		req["webhook"] = webhookURL
	}

	var resp LinkTokenResponse
	if err := c.post(ctx, "/link/token/create", req, &resp); err != nil {
		return LinkTokenResponse{}, err
	}
	return resp, nil
}

func (c *Client) ExchangePublicToken(ctx context.Context, publicToken string) (ExchangePublicTokenResponse, error) {
	req := map[string]string{
		"client_id":    c.clientID,
		"secret":       c.secret,
		"public_token": publicToken,
	}
	var resp ExchangePublicTokenResponse
	if err := c.post(ctx, "/item/public_token/exchange", req, &resp); err != nil {
		return ExchangePublicTokenResponse{}, err
	}
	return resp, nil
}

func (c *Client) SyncTransactions(ctx context.Context, accessToken, cursor string) (TransactionsSyncResponse, error) {
	req := map[string]any{
		"client_id":    c.clientID,
		"secret":       c.secret,
		"access_token": accessToken,
		"count":        100,
	}
	if cursor != "" {
		req["cursor"] = cursor
	}
	var resp TransactionsSyncResponse
	if err := c.post(ctx, "/transactions/sync", req, &resp); err != nil {
		return TransactionsSyncResponse{}, err
	}
	return resp, nil
}

func (c *Client) post(ctx context.Context, path string, payload any, output any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal plaid request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create plaid request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("call plaid sandbox %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		limited, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("plaid sandbox %s returned %s: %s", path, resp.Status, strings.TrimSpace(string(limited)))
	}
	if err := json.NewDecoder(resp.Body).Decode(output); err != nil {
		return fmt.Errorf("decode plaid sandbox response: %w", err)
	}
	return nil
}

