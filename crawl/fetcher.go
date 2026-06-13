package crawl

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/time/rate"
)

// maxBody caps the decompressed response body so one pathological page cannot
// exhaust memory.
const maxBody = 8 << 20 // 8 MiB

// conds carries the validators for a conditional GET, so a re-crawl can ask the
// server whether a page changed instead of refetching it whole.
type conds struct {
	ETag         string
	LastModified string
}

// fetchResult is one HTTP fetch outcome.
type fetchResult struct {
	URL          string
	Status       int
	ContentType  string
	ETag         string
	LastModified string
	Body         []byte
	NotModified  bool // server answered 304 to a conditional GET
}

// fetcher performs paced, retrying HTTP GETs for a single host. The rate limiter
// and crawl-delay together bound how fast the host is hit; transient 429 and 5xx
// responses are retried with exponential backoff that honors Retry-After.
type fetcher struct {
	http    *http.Client
	ua      string
	retries int
	limiter *rate.Limiter
}

// newFetcher builds a fetcher. delay is the minimum gap between requests (the
// larger of the configured rate and the robots crawl-delay); zero means no
// explicit pacing beyond the transport.
func newFetcher(client *http.Client, ua string, retries int, delay time.Duration) *fetcher {
	f := &fetcher{http: client, ua: ua, retries: retries}
	if delay > 0 {
		f.limiter = rate.NewLimiter(rate.Every(delay), 1)
	}
	return f
}

// get fetches one URL, applying pacing, conditional headers, and retry. A 304
// response sets NotModified; 2xx returns the capped body.
func (f *fetcher) get(ctx context.Context, target string, c conds) (*fetchResult, error) {
	var lastErr error
	for attempt := 0; attempt <= f.retries; attempt++ {
		if f.limiter != nil {
			if err := f.limiter.Wait(ctx); err != nil {
				return nil, err
			}
		}
		res, retry, wait, err := f.do(ctx, target, c)
		if err == nil {
			return res, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
		if attempt < f.retries {
			if wait <= 0 {
				wait = backoff(attempt)
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(wait):
			}
		}
	}
	return nil, fmt.Errorf("get %s: %w", target, lastErr)
}

func (f *fetcher) do(ctx context.Context, target string, c conds) (res *fetchResult, retry bool, wait time.Duration, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, false, 0, err
	}
	req.Header.Set("User-Agent", f.ua)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,text/markdown,*/*;q=0.8")
	if c.ETag != "" {
		req.Header.Set("If-None-Match", c.ETag)
	}
	if c.LastModified != "" {
		req.Header.Set("If-Modified-Since", c.LastModified)
	}

	resp, err := f.http.Do(req)
	if err != nil {
		return nil, true, 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	out := &fetchResult{
		URL:          target,
		Status:       resp.StatusCode,
		ContentType:  resp.Header.Get("Content-Type"),
		ETag:         resp.Header.Get("ETag"),
		LastModified: resp.Header.Get("Last-Modified"),
	}

	switch {
	case resp.StatusCode == http.StatusNotModified:
		out.NotModified = true
		return out, false, 0, nil
	case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
		return nil, true, retryAfter(resp), fmt.Errorf("http %d", resp.StatusCode)
	case resp.StatusCode >= 400:
		// A 4xx other than 429 is a permanent answer for this URL; record it.
		return out, false, 0, nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return nil, true, 0, err
	}
	out.Body = body
	return out, false, 0, nil
}

// retryAfter reads a Retry-After header as a delay, returning 0 when absent or
// not a plain seconds count.
func retryAfter(resp *http.Response) time.Duration {
	v := resp.Header.Get("Retry-After")
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil && secs >= 0 {
		return time.Duration(secs) * time.Second
	}
	return 0
}

// backoff is the exponential-with-cap delay before retry attempt n (0-based).
func backoff(attempt int) time.Duration {
	d := time.Duration(1<<uint(attempt)) * 500 * time.Millisecond
	return min(d, 10*time.Second)
}
