package openaicompatible

import (
	"bufio"
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
	client *Client
}

type chatRequest struct {
	Model       string         `json:"model"`
	Messages    []chat.Message `json:"messages"`
	Stream      bool           `json:"stream"`
	Temperature *float64       `json:"temperature,omitempty"`
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

	return &Provider{
		client: NewClient(
			apiKey,
			baseURL,
		),
	}
}

func (p *Provider) Chat(
	ctx context.Context,
	req *chat.ChatCompletionRequest,
) (*chat.ChatCompletionResponse, error) {

	openAIReq := chatRequest{
		Model:       req.Model,
		Messages:    req.Messages,
		Temperature: req.Temperature,
	}

	body, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, fmt.Errorf(
			"marshal provider request: %w",
			err,
		)
	}

	resp, err := p.client.Do(
		ctx,
		body,
	)
	if err != nil {
		return nil, mapProviderRequestError(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {

		var providerError errorResponse

		if err := json.NewDecoder(resp.Body).Decode(&providerError); err != nil {

			return nil, &errors.Error{
				HTTPStatus: http.StatusBadGateway,
				Message: fmt.Sprintf(
					"provider returned HTTP %d",
					resp.StatusCode,
				),
				Type: errors.CodeProviderError,
				Code: "upstream_error",
			}
		}

		message := providerError.Error.Message

		if message == "" {
			message = fmt.Sprintf(
				"provider returned HTTP %d",
				resp.StatusCode,
			)
		}

		return nil, mapProviderError(
			resp.StatusCode,
			message,
			providerError.Error.Code,
		)
	}
	var providerResp chatResponse

	if err := json.NewDecoder(resp.Body).Decode(&providerResp); err != nil {
		return nil, fmt.Errorf(
			"decode provider response: %w",
			err,
		)
	}

	if len(providerResp.Choices) == 0 {
		return nil, fmt.Errorf("provider response contains no choices")
	}

	return &chat.ChatCompletionResponse{
		ID:      providerResp.ID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   providerResp.Model,
		Choices: []chat.Choice{
			{
				Index: providerResp.Choices[0].Index,
				Message: chat.Message{
					Role:    providerResp.Choices[0].Message.Role,
					Content: providerResp.Choices[0].Message.Content,
				},
				FinishReason: providerResp.Choices[0].FinishReason,
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

func (p *Provider) ChatStream(
	ctx context.Context,
	req *chat.ChatCompletionRequest,
	writer provider.StreamWriter,
) error {

	providerReq := chatRequest{
		Model:       req.Model,
		Messages:    req.Messages,
		Stream:      true,
		Temperature: req.Temperature,
	}

	body, err := json.Marshal(providerReq)
	if err != nil {
		return fmt.Errorf(
			"marshal provider stream request: %w",
			err,
		)
	}

	resp, err := p.client.DoStream(
		ctx,
		body,
	)

	fmt.Println(
		"STREAM STATUS:",
		resp.StatusCode,
	)

	if err != nil {
		return mapProviderRequestError(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 ||
		resp.StatusCode >= 300 {

		fmt.Println(
			"STREAM PROVIDER ERROR STATUS:",
			resp.StatusCode,
		)
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

	fmt.Println(">>> handleStreamError")
	var providerError errorResponse

	if err := json.NewDecoder(resp.Body).
		Decode(&providerError); err != nil {

		return &errors.Error{
			HTTPStatus: http.StatusBadGateway,
			Message: fmt.Sprintf(
				"provider returned HTTP %d",
				resp.StatusCode,
			),
			Type: errors.CodeProviderError,
			Code: "upstream_error",
		}
	}

	message := providerError.Error.Message

	if message == "" {
		message = fmt.Sprintf(
			"provider returned HTTP %d",
			resp.StatusCode,
		)
	}

	// return mapProviderError(
	// 	resp.StatusCode,
	// 	message,
	// 	providerError.Error.Code,
	// )
	err := mapProviderError(
		resp.StatusCode,
		message,
		providerError.Error.Code,
	)

	fmt.Printf(
		"STREAM MAPPED ERROR: %#v\n",
		err,
	)

	return err
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

		// fmt.Printf(
		// 	"STREAM LINE: %q\n",
		// 	line,
		// )

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
				"decode provider stream chunk: %w",
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
			"read provider stream: %w",
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
