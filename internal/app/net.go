package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"freewave/internal/migu"
)

var apiClient = &http.Client{Timeout: 35 * time.Second}

// fetchBytes fetches a small resource such as artwork.
func fetchBytes(ctx context.Context, rawURL string) ([]byte, error) {
	if rawURL == "" {
		return nil, errors.New("empty url")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", migu.UserAgent)
	req.Header.Set("Referer", "https://music.migu.cn/")
	res, err := apiClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch: %s", res.Status)
	}
	return io.ReadAll(io.LimitReader(res.Body, 12<<20))
}
