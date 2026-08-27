package openaicompatible

import (
	"bytes"
	"context"
	"net/http"
	"strings"
)

type Client struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func NewClient(
	apiKey string,
	baseURL string,
) *Client {

	return &Client{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		client: &http.Client{
			Timeout: 0,
		},
	}
}

func (c *Client) Do(
	ctx context.Context,
	reqBody []byte,
) (*http.Response, error) {

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/chat/completions",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set(
		"Content-Type",
		"application/json",
	)

	req.Header.Set(
		"Authorization",
		"Bearer "+c.apiKey,
	)

	return c.client.Do(req)
}

func (c *Client) DoStream(
	ctx context.Context,
	reqBody []byte,
) (*http.Response, error) {

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/chat/completions",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set(
		"Content-Type",
		"application/json",
	)

	req.Header.Set(
		"Authorization",
		"Bearer "+c.apiKey,
	)

	req.Header.Set(
		"Accept",
		"text/event-stream",
	)

	return c.client.Do(req)
}
