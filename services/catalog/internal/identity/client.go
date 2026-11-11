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

var ErrUnauthenticated = errors.New("invalid session")
var ErrUnavailable = errors.New("identity verification unavailable")

type Client struct {
	meURL string
	http  *http.Client
}

func NewClient(baseURL string) (*Client, error) {
	base, err := url.Parse(baseURL)
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") || base.Hostname() == "" ||
		base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return nil, errors.New("invalid IDENTITY_BASE_URL")
	}

	return &Client{
		meURL: strings.TrimRight(baseURL, "/") + "/api/v1/identity/me",
		http: &http.Client{
			Timeout: 3 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

func (c *Client) Role(ctx context.Context, token, requestID string) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.meURL, nil)
	if err != nil {
		return "", ErrUnavailable
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("X-Request-ID", requestID)

	response, err := c.http.Do(request)
	if err != nil {
		return "", ErrUnavailable
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusUnauthorized:
		return "", ErrUnauthenticated
	case http.StatusOK:
	default:
		return "", ErrUnavailable
	}

	const maxResponseBytes = 16 * 1024
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil || len(body) > maxResponseBytes {
		return "", ErrUnavailable
	}

	var profile struct {
		Data struct {
			ID   string `json:"id"`
			Role string `json:"role"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &profile); err != nil || strings.TrimSpace(profile.Data.ID) == "" {
		return "", ErrUnavailable
	}

	switch profile.Data.Role {
	case "OWNER", "ADMIN", "CUSTOMER":
		return profile.Data.Role, nil
	default:
		return "", ErrUnavailable
	}
}
