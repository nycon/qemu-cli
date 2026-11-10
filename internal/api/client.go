// Package api is the HTTP client for the QEMU Interface panel API.
package api

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nycon/qemu-cli/internal/config"
)

const timeout = 30 * time.Second

// Client talks to /api/v1 with X-API-Key auth.
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// NewFromActive builds a client from the active CLI context.
func NewFromActive() (*Client, error) {
	_, ctx, err := config.Active()
	if err != nil {
		return nil, err
	}
	return New(ctx.ServerURL, ctx.Token, ctx.Insecure), nil
}

// New creates a client for the given server.
func New(serverURL, token string, insecure bool) *Client {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	if insecure {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // explicit --insecure for self-signed panels
	}
	return &Client{
		BaseURL: strings.TrimRight(serverURL, "/") + "/api/v1",
		Token:   token,
		HTTPClient: &http.Client{
			Timeout:   timeout,
			Transport: tr,
		},
	}
}

// APIError is a non-2xx API response.
type APIError struct {
	Status int
	Detail string
}

func (e *APIError) Error() string {
	if e.Status == 401 {
		return "authentication failed — check your API token\n  qemu-cli context create <name> --url <url> --token <qi_...>"
	}
	return fmt.Sprintf("API error (%d): %s", e.Status, e.Detail)
}

func (c *Client) do(method, path string, query url.Values, body any) (json.RawMessage, error) {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rdr = bytes.NewReader(b)
	}
	u := c.BaseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequest(method, u, rdr)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", c.Token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNoContent {
		return json.RawMessage("{}"), nil
	}
	if resp.StatusCode >= 400 {
		detail := string(data)
		var parsed map[string]any
		if json.Unmarshal(data, &parsed) == nil {
			if d, ok := parsed["detail"]; ok {
				b, _ := json.Marshal(d)
				detail = string(b)
			}
		}
		return nil, &APIError{Status: resp.StatusCode, Detail: detail}
	}
	if len(data) == 0 {
		return json.RawMessage("{}"), nil
	}
	return json.RawMessage(data), nil
}

func (c *Client) Get(path string, query url.Values) (json.RawMessage, error) {
	return c.do(http.MethodGet, path, query, nil)
}

func (c *Client) Post(path string, body any) (json.RawMessage, error) {
	return c.do(http.MethodPost, path, nil, body)
}

func (c *Client) Put(path string, body any) (json.RawMessage, error) {
	return c.do(http.MethodPut, path, nil, body)
}

func (c *Client) Patch(path string, body any) (json.RawMessage, error) {
	return c.do(http.MethodPatch, path, nil, body)
}

func (c *Client) Delete(path string) (json.RawMessage, error) {
	return c.do(http.MethodDelete, path, nil, nil)
}

// Decode unmarshals raw JSON into dest.
func Decode(raw json.RawMessage, dest any) error {
	return json.Unmarshal(raw, dest)
}

// AsList returns a JSON array (unwraps {"items":[...]} when present).
func AsList(raw json.RawMessage) ([]json.RawMessage, error) {
	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr, nil
	}
	var wrap struct {
		Items []json.RawMessage `json:"items"`
		Total int               `json:"total"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return nil, err
	}
	return wrap.Items, nil
}
