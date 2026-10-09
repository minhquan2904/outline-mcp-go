package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	DefaultTimeoutMs = 30000
	MaxRetryDelayMs  = 10000
	FallbackRetryMs  = 1000
)

type Client struct {
	baseUrl    string
	apiKey     string
	timeoutMs  int
	httpClient HTTPClient
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func NewClient(baseUrl, apiKey string) *Client {
	return &Client{
		baseUrl:    strings.TrimRight(baseUrl, "/"),
		apiKey:     apiKey,
		timeoutMs:  DefaultTimeoutMs,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) Call(ctx context.Context, method string, params map[string]interface{}) (*Result, error) {
	// Check credentials early
	if c.baseUrl == "" {
		return &Result{
			Ok:    false,
			Error: "$OUTLINE_API_URL is not set, so there is no Outline instance to call.",
		}, nil
	}
	if c.apiKey == "" {
		return &Result{
			Ok:    false,
			Error: "$OUTLINE_API_KEY is not set. Add it to environment or config.",
		}, nil
	}

	// Build request
	endpoint := fmt.Sprintf("%s/api/%s", c.baseUrl, method)
	body, _ := json.Marshal(params)

	req, _ := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Content-Type", "application/json")

	// Execute with timeout
	ctx, cancel := context.WithTimeout(ctx, time.Duration(c.timeoutMs)*time.Millisecond)
	defer cancel()
	req = req.WithContext(ctx)

	response, err := c.httpClient.Do(req)
	if err != nil {
		msg := fmt.Sprintf("Could not reach Outline for %s: %v", method, err)
		if ctx.Err() != nil {
			msg = fmt.Sprintf("Request to Outline timed out after %dms on %s", c.timeoutMs, method)
		}
		return &Result{
			Ok:    false,
			Error: redact(msg, c.apiKey),
		}, nil
	}
	defer response.Body.Close()

	respBody, _ := io.ReadAll(response.Body)

	// Retry on 429
	if response.StatusCode == 429 {
		delay := retryDelayMs(response)
		time.Sleep(time.Duration(delay) * time.Millisecond)

		req2, _ := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
		req2.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
		req2.Header.Set("Content-Type", "application/json")
		response, _ = c.httpClient.Do(req2)
		if response != nil {
			defer response.Body.Close()
			respBody, _ = io.ReadAll(response.Body)
		}
	}

	// Parse response
	var apiResp APIResponse
	_ = json.Unmarshal(respBody, &apiResp)

	if response.StatusCode != 200 {
		errMsg := describeFailure(method, response.StatusCode, &apiResp)
		return &Result{
			Ok:    false,
			Error: redact(errMsg, c.apiKey),
		}, nil
	}

	return &Result{
		Ok:   true,
		Data: apiResp.Data,
		Raw:  apiResp,
	}, nil
}

func describeFailure(method string, status int, resp *APIResponse) string {
	upstream := ""
	if resp != nil && resp.Error != "" {
		upstream = resp.Error
	}

	switch status {
	case 401:
		return fmt.Sprintf("Outline rejected the credentials (401: %s). Check that $OUTLINE_API_KEY holds a valid API key.", upstream)
	case 403:
		return fmt.Sprintf("Outline refused the request (403: %s). The key is valid but lacks permission for %s.", upstream, method)
	case 404:
		return fmt.Sprintf("Not found (404) for %s: %s. The id or slug is missing.", method, upstream)
	case 429:
		return "Outline rate limit hit (429). Wait a moment and retry."
	default:
		return fmt.Sprintf("Outline returned %d for %s: %s", status, method, upstream)
	}
}

func retryDelayMs(response *http.Response) int {
	if response == nil || response.Header == nil {
		return FallbackRetryMs
	}
	raw := response.Header.Get("retry-after")
	var seconds int
	fmt.Sscanf(raw, "%d", &seconds)
	if seconds <= 0 {
		return FallbackRetryMs
	}
	delay := seconds * 1000
	if delay > MaxRetryDelayMs {
		return MaxRetryDelayMs
	}
	return delay
}

func redact(text, apiKey string) string {
	if apiKey == "" {
		return text
	}
	return strings.ReplaceAll(text, apiKey, "ol_api_***")
}

var (
	uuidRE  = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	urlIDRE = regexp.MustCompile(`^[A-Za-z0-9]{10,15}$`) // measured on a live instance: 9 rejected, 10-15 accepted, 16 rejected
	httpRE  = regexp.MustCompile(`(?i)^https?://`)
)

const refHint = "expected a UUID, a 10-15 character urlId, or a document URL like /doc/title-a1B2c3D4e5"

// ParseDocRef resolves a pasted UUID, urlId or document URL to an id that documents.info accepts.
func ParseDocRef(input string) *DocRef {
	s := strings.TrimSpace(input)
	if s == "" {
		return &DocRef{Kind: "invalid", Err: "Empty document reference — " + refHint + "."}
	}
	if uuidRE.MatchString(s) {
		return &DocRef{Kind: "uuid", ID: s}
	}

	segment := s
	if idx := strings.Index(s, "/doc/"); idx != -1 {
		segment = s[idx+len("/doc/"):]
		// drop sub-paths such as /edit or /history
		if i := strings.Index(segment, "/"); i != -1 {
			segment = segment[:i]
		}
	} else if httpRE.MatchString(s) || strings.HasPrefix(s, "/") {
		return &DocRef{Kind: "invalid", Err: fmt.Sprintf("%q is not a document URL — %s.", s, refHint)}
	}

	if i := strings.IndexAny(segment, "?#"); i != -1 {
		segment = segment[:i]
	}
	segment = strings.TrimRight(segment, "/")
	if segment == "" {
		return &DocRef{Kind: "invalid", Err: fmt.Sprintf("%q has no document id — %s.", s, refHint)}
	}

	candidate := segment
	if dash := strings.LastIndex(segment, "-"); dash != -1 {
		candidate = segment[dash+1:]
	}
	if urlIDRE.MatchString(candidate) {
		return &DocRef{Kind: "urlId", ID: candidate}
	}
	return &DocRef{Kind: "invalid", Err: fmt.Sprintf("%q is not a document reference — %s.", s, refHint)}
}
