package netutil

import (
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ResilientClient is a wrapper around http.Client that implements exponential backoff and retries.
type ResilientClient struct {
	client  *http.Client
	retries int
	baseMs  time.Duration
	capMs   time.Duration
}

func NewResilientClient(timeout time.Duration, retries int, resolver string) *ResilientClient {
	client := &http.Client{Timeout: timeout}
	if resolver != "" {
		client.Transport = NewDoHTransport(resolver)
	}

	return &ResilientClient{
		client:  client,
		retries: retries,
		baseMs:  500 * time.Millisecond,
		capMs:   20 * time.Second,
	}
}

func (c *ResilientClient) Do(req *http.Request) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= c.retries; attempt++ {
		if req.Context().Err() != nil {
			return nil, req.Context().Err()
		}

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = err
			if attempt < c.retries {
				time.Sleep(c.backoff(attempt, 0))
				continue
			}
			return nil, err
		}

		// Check for retryable HTTP status codes
		switch resp.StatusCode {
		case 408, 425, 429, 500, 502, 503, 504:
			server := strings.ToLower(resp.Header.Get("Server"))
			if resp.StatusCode == 503 && (strings.Contains(server, "ddos-guard") || strings.Contains(server, "cloudflare")) {
				resp.Body.Close()
				return nil, fmt.Errorf("request blocked by %s (HTTP %d)", server, resp.StatusCode)
			}

			if attempt >= c.retries {
				resp.Body.Close()
				return nil, fmt.Errorf("request failed after %d retries (HTTP %d)", c.retries, resp.StatusCode)
			}

			retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
			resp.Body.Close()
			time.Sleep(c.backoff(attempt, retryAfter))
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("exhausted retries, last error: %v", lastErr)
}

func (c *ResilientClient) backoff(attempt int, retryAfter time.Duration) time.Duration {
	exp := float64(c.baseMs) * float64(int(1)<<attempt)
	if exp > float64(c.capMs) {
		exp = float64(c.capMs)
	}
	jittered := time.Duration(rand.Float64() * exp)

	if retryAfter > 0 && retryAfter > jittered {
		return retryAfter
	}
	return jittered
}

func parseRetryAfter(val string) time.Duration {
	if val == "" {
		return 0
	}
	val = strings.TrimSpace(val)
	if sec, err := strconv.Atoi(val); err == nil {
		return time.Duration(sec) * time.Second
	}

	if t, err := time.Parse(time.RFC1123, val); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d
		}
	}
	return 0
}
