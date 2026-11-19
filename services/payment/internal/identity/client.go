package identity

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	ErrUnauthenticated = errors.New("invalid customer session")
	ErrUnavailable     = errors.New("identity is unavailable")
)

type Client struct {
	meURL string
	http  *http.Client
}

func NewClient(baseURL string) (*Client, error) {
	if baseURL == "" {
		baseURL = "http://localhost:8081"
	}
	base, err := url.Parse(baseURL)
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") || base.Hostname() == "" ||
		base.User != nil || base.RawQuery != "" || base.ForceQuery || base.Fragment != "" ||
		(base.Path != "" && base.Path != "/") {
		return nil, errors.New("IDENTITY_BASE_URL must be an HTTP origin")
	}
	return &Client{
		meURL: strings.TrimRight(base.String(), "/") + "/api/v1/identity/me",
		http: &http.Client{
			Timeout: 2 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

type Profile struct {
	ID       string `json:"id"`
	Role     string `json:"role"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
}

func (c *Client) Profile(ctx context.Context, token, requestID string) (Profile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.meURL, nil)
	if err != nil {
		return Profile{}, ErrUnavailable
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Request-ID", requestID)
	res, err := c.http.Do(req)
	if err != nil {
		return Profile{}, ErrUnavailable
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusUnauthorized {
		return Profile{}, ErrUnauthenticated
	}
	if res.StatusCode != http.StatusOK {
		return Profile{}, ErrUnavailable
	}

	const maxResponseBytes = 16 * 1024
	body, err := io.ReadAll(io.LimitReader(res.Body, maxResponseBytes+1))
	if err != nil || len(body) > maxResponseBytes {
		return Profile{}, ErrUnavailable
	}
	var profile struct {
		Data Profile `json:"data"`
	}
	if json.Unmarshal(body, &profile) != nil || strings.TrimSpace(profile.Data.ID) == "" || strings.TrimSpace(profile.Data.Role) == "" {
		return Profile{}, ErrUnavailable
	}
	return profile.Data, nil
}
