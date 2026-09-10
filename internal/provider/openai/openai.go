package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ai_gateway/internal/chat"
	"ai_gateway/internal/errors"
	"ai_gateway/internal/provider"
)

var _ provider.ModelProvider = (*Provider)(nil)

type Provider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

type chatRequest struct {
	Model    string         `json:"model"`
	Messages []chat.Message `json:"messages"`
	Stream   bool           `json:"stream"`
}

type chatResponse struct {
	ID     string `json:"id"`
	Object string `json:"object"`
	Model  string `json:"model"`

	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

func NewProvider(
	apiKey string,
	baseURL string,
) *Provider {

	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}

	return &Provider{
		apiKey: apiKey,
		baseURL: strings.TrimRight(
			baseURL,
			"/",
		),
		client: &http.Client{
			Timeout: 0,
		},
	}
}

func (p *Provider) Chat(
	ctx context.Context,
	req *chat.ChatCompletionRequest,
) (*chat.ChatCompletionResponse, error) {

	openAIReq := chatRequest{
		Model:    req.Model,
		Messages: req.Messages,
	}

	body, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, fmt.Errorf("marshal openai request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		p.baseURL+"/v1/chat/completions",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("create openai request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call openai api: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {

		var openAIError errorResponse

		if err := json.NewDecoder(resp.Body).Decode(&openAIError); err != nil {

			return nil, &errors.Error{
				HTTPStatus: http.StatusBadGateway,
				Message: fmt.Sprintf(
					"OpenAI returned HTTP %d",
					resp.StatusCode,
				),
				Type: errors.CodeProviderError,
				Code: "upstream_error",
			}
		}

		message := openAIError.Error.Message

		if message == "" {
			message = fmt.Sprintf(
				"OpenAI returned HTTP %d",
				resp.StatusCode,
			)
		}

		return nil, mapOpenAIError(
			resp.StatusCode,
			message,
			openAIError.Error.Code,
		)
	}
	var openAIResp chatResponse

	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		return nil, fmt.Errorf(
			"decode openai response: %w",
			err,
		)
	}

	if len(openAIResp.Choices) == 0 {
		return nil, fmt.Errorf("openai response contains no choices")
	}
	content := openAIResp.Choices[0].Message.Content

	return &chat.ChatCompletionResponse{
		ID:      openAIResp.ID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   openAIResp.Model,
		Choices: []chat.Choice{
			{
				Index: openAIResp.Choices[0].Index,
				Message: chat.Message{
					Role:    openAIResp.Choices[0].Message.Role,
					Content: &content,
				},
				FinishReason: openAIResp.Choices[0].FinishReason,
			},
		},
	}, nil
}

type errorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

func mapOpenAIError(
	status int,
	message string,
	code string,
) *errors.Error {

	switch status {

	case http.StatusUnauthorized:
		return &errors.Error{
			HTTPStatus: http.StatusUnauthorized,
			Message:    message,
			Type:       errors.CodeAuthentication,
			Code:       code,
		}

	case http.StatusForbidden:
		return &errors.Error{
			HTTPStatus: http.StatusForbidden,
			Message:    message,
			Type:       errors.CodePermissionDenied,
			Code:       code,
		}

	case http.StatusNotFound:
		return &errors.Error{
			HTTPStatus: http.StatusNotFound,
			Message:    message,
			Type:       errors.CodeNotFound,
			Code:       code,
		}

	case http.StatusTooManyRequests:
		return &errors.Error{
			HTTPStatus: http.StatusTooManyRequests,
			Message:    message,
			Type:       errors.CodeRateLimit,
			Code:       code,
		}

	case http.StatusRequestTimeout:
		return &errors.Error{
			HTTPStatus: http.StatusGatewayTimeout,
			Message:    message,
			Type:       errors.CodeProviderTimeout,
			Code:       code,
		}

	default:
		return &errors.Error{
			HTTPStatus: http.StatusBadGateway,
			Message:    message,
			Type:       errors.CodeProviderError,
			Code:       code,
		}
	}
}

func (p *Provider) ChatStream(
	ctx context.Context,
	req *chat.ChatCompletionRequest,
	writer provider.StreamWriter,
) error {

	openAIReq := chatRequest{
		Model:    req.Model,
		Messages: req.Messages,
		Stream:   true,
	}

	body, err := json.Marshal(openAIReq)
	if err != nil {
		return fmt.Errorf(
			"marshal openai stream request: %w",
			err,
		)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		p.baseURL+"/v1/chat/completions",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf(
			"create openai stream request: %w",
			err,
		)
	}

	httpReq.Header.Set(
		"Content-Type",
		"application/json",
	)

	httpReq.Header.Set(
		"Authorization",
		"Bearer "+p.apiKey,
	)

	httpReq.Header.Set(
		"Accept",
		"text/event-stream",
	)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf(
			"call openai stream api: %w",
			err,
		)
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return p.handleStreamError(resp)
	}

	return p.readSSE(
		ctx,
		resp.Body,
		req,
		writer,
	)
}

func (p *Provider) handleStreamError(
	resp *http.Response,
) error {

	var openAIError errorResponse

	if err := json.NewDecoder(resp.Body).
		Decode(&openAIError); err != nil {

		return &errors.Error{
			HTTPStatus: http.StatusBadGateway,
			Message: fmt.Sprintf(
				"OpenAI returned HTTP %d",
				resp.StatusCode,
			),
			Type: errors.CodeProviderError,
			Code: "upstream_error",
		}
	}

	message := openAIError.Error.Message

	if message == "" {
		message = fmt.Sprintf(
			"OpenAI returned HTTP %d",
			resp.StatusCode,
		)
	}

	return mapOpenAIError(
		resp.StatusCode,
		message,
		openAIError.Error.Code,
	)
}

func (p *Provider) readSSE(
	ctx context.Context,
	body io.Reader,
	req *chat.ChatCompletionRequest,
	writer provider.StreamWriter,
) error {

	scanner := bufio.NewScanner(body)

	scanner.Buffer(
		make([]byte, 64*1024),
		1024*1024,
	)

	for scanner.Scan() {

		line := scanner.Text()

		if line == "" {
			continue
		}

		if !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimSpace(
			strings.TrimPrefix(line, "data:"),
		)

		if data == "[DONE]" {
			return nil
		}

		var chunk chatResponseChunk

		if err := json.Unmarshal(
			[]byte(data),
			&chunk,
		); err != nil {

			return fmt.Errorf(
				"decode openai stream chunk: %w",
				err,
			)
		}

		gatewayChunk := convertChunk(
			&chunk,
			req.Model,
		)

		if err := writer(gatewayChunk); err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()

		default:
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf(
			"read openai stream: %w",
			err,
		)
	}

	return nil
}

type chatResponseChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`

	Choices []struct {
		Index int `json:"index"`

		Delta struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"delta"`

		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

func convertChunk(
	chunk *chatResponseChunk,
	model string,
) *chat.ChatCompletionChunk {

	result := &chat.ChatCompletionChunk{
		ID:      chunk.ID,
		Object:  "chat.completion.chunk",
		Created: chunk.Created,
		Model:   model,
		Choices: make([]chat.ChunkChoice, 0, len(chunk.Choices)),
	}

	for _, choice := range chunk.Choices {

		result.Choices = append(
			result.Choices,
			chat.ChunkChoice{
				Index: choice.Index,
				Delta: chat.Delta{
					Role:    choice.Delta.Role,
					Content: choice.Delta.Content,
				},
				FinishReason: choice.FinishReason,
			},
		)
	}

	return result
}
