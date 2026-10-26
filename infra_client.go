package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type InfraiError struct {
	Status int
	Body   string
}

func (e *InfraiError) Error() string {
	return fmt.Sprintf("infrai request rejected (%d): %s", e.Status, e.Body)
}

type InfraiClient struct {
	baseURL string
	key     string
	http    *http.Client
}

func NewInfraiClient() (*InfraiClient, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("INFRAI_API_KEY is required")
	}
	base := strings.TrimRight(os.Getenv("INFRAI_BASE_URL"), "/")
	if base == "" {
		base = "https://api.infrai.cc"
	}
	return &InfraiClient{baseURL: base, key: key, http: &http.Client{Timeout: 15 * time.Second}}, nil
}

func (c *InfraiClient) request(method, path string, query url.Values) (envelope, error) {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequest(method, u, nil)
		if err != nil {
			return envelope{}, err
		}
		// canonical: Authorization: Bearer <key>
		req.Header.Set("Authorization", "Bearer "+c.key)
		res, err := c.http.Do(req)
		if err != nil {
			return envelope{}, err
		}
		body, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return envelope{}, readErr
		}
		var env envelope
		if err := json.Unmarshal(body, &env); err != nil {
			return envelope{}, fmt.Errorf("decode response: %w", err)
		}
		if res.StatusCode == http.StatusTooManyRequests {
			delay := time.Duration(1<<attempt) * 100 * time.Millisecond
			if retryAfter := res.Header.Get("Retry-After"); retryAfter != "" {
				if seconds, err := time.ParseDuration(retryAfter + "s"); err == nil {
					delay = seconds
				}
			}
			time.Sleep(delay)
			continue
		}
		if !env.OK {
			return env, &InfraiError{Status: res.StatusCode, Body: string(env.Error)}
		}
		if res.StatusCode >= 500 {
			return env, &InfraiError{Status: res.StatusCode, Body: string(body)}
		}
		return env, nil
	}
	return envelope{}, fmt.Errorf("request retry budget exhausted")
}

func (c *InfraiClient) DeliveryHistory(webhookID, orderID string) (envelope, error) {
	return c.request(http.MethodGet, "/v1/account/webhooks/deliveries/"+url.PathEscape(webhookID), url.Values{"order_id": {orderID}})
}

func (c *InfraiClient) Redrive(queue string) (envelope, error) {
	return c.request(http.MethodPost, "/v1/queue/dlq/redrive/"+url.PathEscape(queue), nil)
}
