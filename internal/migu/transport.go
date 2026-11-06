package migu

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var apiClient = &http.Client{Timeout: 35 * time.Second}

// flexString reads a field that the Migu API may encode as a string, a
// number or a list of strings.
type flexString string

func (s *flexString) UnmarshalJSON(b []byte) error {
	b = []byte(strings.TrimSpace(string(b)))
	if len(b) == 0 || string(b) == "null" {
		*s = ""
		return nil
	}
	switch b[0] {
	case '"':
		var v string
		if err := json.Unmarshal(b, &v); err != nil {
			return err
		}
		*s = flexString(v)
	case '[':
		var v []string
		if err := json.Unmarshal(b, &v); err != nil {
			return err
		}
		*s = flexString(strings.Join(v, ", "))
	default:
		*s = flexString(string(b))
	}
	return nil
}

var tagRE = regexp.MustCompile(`<[^>]*>`)
var spaceRE = regexp.MustCompile(`\s+`)

func cleanDisplay(s string) string {
	s = html.UnescapeString(strings.TrimSpace(s))
	s = spaceRE.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func cleanDescription(s string) string {
	s = tagRE.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	s = spaceRE.ReplaceAllString(s, " ")
	s = strings.TrimSpace(s)
	runes := []rune(s)
	if len(runes) > 900 {
		s = string(runes[:900]) + "…"
	}
	return s
}

// fetchBytes fetches a small resource such as artwork.
func fetchBytes(ctx context.Context, rawURL string) ([]byte, error) {
	if rawURL == "" {
		return nil, errors.New("empty url")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", miguUA)
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
