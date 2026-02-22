package client

import (
	"encoding/base64"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

const (
	maxRetries     = 1
	baseBackoffSec = 1
)

// Client is an authenticated HTTP client for the Mixpanel API.
type Client struct {
	httpClient *http.Client
	auth       string // base64-encoded "user:secret"
	region     string // us, eu, in
	projectID  string
	debug      bool
}

// New creates a Client. serviceAccount and serviceSecret are used for Basic Auth.
// region must be one of "us", "eu", "in". debug enables request/response logging.
func New(serviceAccount, serviceSecret, region, projectID string, debug bool) (*Client, error) {
	if !ValidRegion(region) {
		return nil, fmt.Errorf("invalid region %q; must be one of: us, eu, in", region)
	}
	if serviceAccount == "" || serviceSecret == "" {
		return nil, fmt.Errorf("service_account and service_secret must be configured; run: mp config set service_account <value>")
	}

	auth := base64.StdEncoding.EncodeToString([]byte(serviceAccount + ":" + serviceSecret))

	return &Client{
		httpClient: &http.Client{Timeout: 120 * time.Second},
		auth:       auth,
		region:     region,
		projectID:  projectID,
		debug:      debug,
	}, nil
}

// Get performs an authenticated GET request against the given API family and path.
// params are appended as query parameters.
func (c *Client) Get(apiFamily, path string, params url.Values) (*http.Response, error) {
	return c.do(http.MethodGet, apiFamily, path, params, nil)
}

// Post performs an authenticated POST request with form-encoded params as the body.
func (c *Client) Post(apiFamily, path string, params url.Values) (*http.Response, error) {
	return c.do(http.MethodPost, apiFamily, path, nil, params)
}

func (c *Client) do(method, apiFamily, path string, query url.Values, form url.Values) (*http.Response, error) {
	base, err := ResolveURL(apiFamily, c.region)
	if err != nil {
		return nil, err
	}

	fullURL := base + path
	if query != nil && len(query) > 0 {
		fullURL += "?" + query.Encode()
	}

	var body io.Reader
	var contentType string
	if form != nil && len(form) > 0 {
		encoded := form.Encode()
		body = stringReader(encoded)
		contentType = "application/x-www-form-urlencoded"
	}

	var resp *http.Response
	for attempt := 0; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest(method, fullURL, body)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}

		req.Header.Set("Authorization", "Basic "+c.auth)
		req.Header.Set("Accept-Encoding", "gzip")
		req.Header.Set("Accept", "application/json")
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}

		c.debugf("--> %s %s\n", method, fullURL)

		resp, err = c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("executing request: %w", err)
		}

		c.debugf("<-- %d %s\n", resp.StatusCode, resp.Status)

		if resp.StatusCode != http.StatusTooManyRequests {
			break
		}

		// Rate limited: back off and retry.
		if attempt < maxRetries {
			wait := backoff(attempt, resp)
			c.debugf("    rate limited, retrying in %v\n", wait)
			resp.Body.Close()
			time.Sleep(wait)
			// Reset body reader for POST retries.
			if form != nil && len(form) > 0 {
				body = stringReader(form.Encode())
			}
		}
	}

	return resp, nil
}

// backoff calculates the wait duration after a 429 response.
// It uses the Retry-After header if present, otherwise exponential backoff.
func backoff(attempt int, resp *http.Response) time.Duration {
	if ra := resp.Header.Get("Retry-After"); ra != "" {
		if secs, err := strconv.Atoi(ra); err == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return time.Duration(math.Pow(2, float64(attempt))*baseBackoffSec) * time.Second
}

func (c *Client) debugf(format string, a ...any) {
	if c.debug {
		fmt.Fprintf(os.Stderr, "[mp debug] "+format, a...)
	}
}

// ProjectID returns the configured project ID.
func (c *Client) ProjectID() string {
	return c.projectID
}

// stringReader creates an io.Reader from a string.
type stringReaderType struct{ s string }

func (r *stringReaderType) Read(p []byte) (int, error) {
	n := copy(p, r.s)
	r.s = r.s[n:]
	if len(r.s) == 0 {
		return n, io.EOF
	}
	return n, nil
}

func stringReader(s string) io.Reader {
	return &stringReaderType{s: s}
}
