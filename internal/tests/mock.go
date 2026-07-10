package tests

import (
	"net/http"
	"net/http/httptest"
	"net/url"
)

type clientInjector interface {
	InjectClient(c *http.Client)
}

type mockTransport struct {
	mockURL *url.URL
	rt      http.RoundTripper
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Rewrite the request to hit the mock server instead of the hardcoded domain
	req.URL.Scheme = m.mockURL.Scheme
	req.URL.Host = m.mockURL.Host
	return m.rt.RoundTrip(req)
}

func newMockClient(ts *httptest.Server) *http.Client {
	u, _ := url.Parse(ts.URL)
	return &http.Client{
		Transport: &mockTransport{
			mockURL: u,
			rt:      ts.Client().Transport,
		},
	}
}
