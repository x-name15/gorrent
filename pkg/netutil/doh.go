package netutil

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var defaultDoHResolvers = map[string]string{
	"cloudflare": "1.1.1.1",
	"google":     "8.8.8.8",
	"quad9":      "9.9.9.9",
	"opendns":    "208.67.222.222",
}

type dohAnswer struct {
	Type int    `json:"type"`
	Data string `json:"data"`
	TTL  int    `json:"TTL"`
}

type dohResponse struct {
	Answer []dohAnswer `json:"Answer"`
}

type cachedDNS struct {
	ips     []string
	expires time.Time
}

var (
	dnsCache = sync.Map{}
	A        = 1
	AAAA     = 28
)

// NewDoHTransport creates an http.Transport that resolves hostnames using DoH.
func NewDoHTransport(resolver string) *http.Transport {
	if resolver == "" {
		return http.DefaultTransport.(*http.Transport).Clone()
	}

	ip := resolver
	if mapped, ok := defaultDoHResolvers[strings.ToLower(resolver)]; ok {
		ip = mapped
	}

	// Just checking if it's a valid IP, else we assume it's already an IP.
	// We only support DoH providers that accept IP directly (Cloudflare, Google, etc.)

	path := "dns-query"
	if ip == "8.8.8.8" || ip == "8.8.4.4" {
		path = "resolve"
	}

	dohEndpoint := fmt.Sprintf("https://%s/%s", ip, path)

	t := http.DefaultTransport.(*http.Transport).Clone()
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	t.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}

		ips, err := resolveViaDoH(ctx, dohEndpoint, host)
		if err != nil || len(ips) == 0 {
			// fallback to system DNS
			return dialer.DialContext(ctx, network, addr)
		}

		var lastErr error
		for _, ip := range ips {
			conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip, port))
			if err == nil {
				return conn, nil
			}
			lastErr = err
		}
		return nil, lastErr
	}

	return t
}

func resolveViaDoH(ctx context.Context, endpoint, host string) ([]string, error) {
	if val, ok := dnsCache.Load(host); ok {
		cached := val.(cachedDNS)
		if time.Now().Before(cached.expires) {
			return cached.ips, nil
		}
	}

	// Try A record first
	ips, ttl, err := queryDoH(ctx, endpoint, host, A)
	if err == nil && len(ips) > 0 {
		dnsCache.Store(host, cachedDNS{
			ips:     ips,
			expires: time.Now().Add(time.Duration(ttl) * time.Second),
		})
		return ips, nil
	}

	// Try AAAA
	ips, ttl, err = queryDoH(ctx, endpoint, host, AAAA)
	if err == nil && len(ips) > 0 {
		dnsCache.Store(host, cachedDNS{
			ips:     ips,
			expires: time.Now().Add(time.Duration(ttl) * time.Second),
		})
		return ips, nil
	}

	return nil, fmt.Errorf("could not resolve %s via DoH", host)
}

func queryDoH(ctx context.Context, endpoint, host string, rrtype int) ([]string, int, error) {
	q := url.QueryEscape(host)
	tStr := "A"
	if rrtype == AAAA {
		tStr = "AAAA"
	}
	u := fmt.Sprintf("%s?name=%s&type=%s", endpoint, q, tStr)

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("accept", "application/dns-json")

	// Must use a separate client to avoid loop!
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("DoH returned %d", resp.StatusCode)
	}

	var jsonResp dohResponse
	if err := json.NewDecoder(resp.Body).Decode(&jsonResp); err != nil {
		return nil, 0, err
	}

	var ips []string
	minTTL := 60
	for _, a := range jsonResp.Answer {
		if a.Type == rrtype {
			ips = append(ips, a.Data)
			if a.TTL > 0 && a.TTL < minTTL {
				minTTL = a.TTL
			}
		}
	}

	return ips, minTTL, nil
}
