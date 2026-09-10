package service

import (
	"context"
	"strings"

	"ai_gateway/internal/chat"
	"ai_gateway/internal/conversation"
	"ai_gateway/internal/modelrouter"
	"ai_gateway/internal/provider"
	"ai_gateway/internal/utils"
)

type ChatService struct {
	router              *modelrouter.Router
	conversationService *ConversationService
}

func NewChatService(
	router *modelrouter.Router,
	conversationService *ConversationService,
) *ChatService {
	return &ChatService{
		router:              router,
		conversationService: conversationService,
	}
}

func (s *ChatService) Chat(
	ctx context.Context,
	req *chat.ChatCompletionRequest,
) (*chat.ChatCompletionResponse, error) {

	p, model, err := s.router.Resolve(req.Model)

	if err != nil {
		return nil, err
	}

	messages, err := s.buildMessages(req)

	if err != nil {
		return nil, err
	}

	providerReq := *req

	providerReq.Model = model
	providerReq.Messages = messages

	response, err := p.Chat(
		ctx,
		&providerReq,
	)

	if err != nil {
		return nil, err
	}

	if req.ConversationID != "" {

		for _, message := range req.Messages {
			err := s.conversationService.AddMessage(
				req.ConversationID,
				conversation.Message{
					Role:    message.Role,
					Content: utils.StringValue(message.Content),
				},
			)

			if err != nil {
				return nil, err
			}
		}

		if len(response.Choices) > 0 {

			message := response.Choices[0].Message

			err := s.conversationService.AddMessage(
				req.ConversationID,
				conversation.Message{
					Role:    message.Role,
					Content: utils.StringValue(message.Content),
				},
			)

			if err != nil {
				return nil, err
			}
		}
	}

	return response, nil
}

func (s *ChatService) ChatStream(
	ctx context.Context,
	req *chat.ChatCompletionRequest,
	writer provider.StreamWriter,
) error {

	p, model, err := s.router.Resolve(req.Model)

	if err != nil {
		return err
	}

	messages, err := s.buildMessages(req)

	if err != nil {
		return err
	}

	providerReq := *req

	providerReq.Model = model
	providerReq.Messages = messages

	var assistantContent strings.Builder

	err = p.ChatStream(
		ctx,
		&providerReq,
		func(chunk *chat.ChatCompletionChunk) error {

			for _, choice := range chunk.Choices {
				assistantContent.WriteString(
					choice.Delta.Content,
				)
			}

			return writer(chunk)
		},
	)

	if err != nil {
		return err
	}

	if req.ConversationID != "" {

		for _, message := range req.Messages {

			err := s.conversationService.AddMessage(
				req.ConversationID,
				conversation.Message{
					Role:    message.Role,
					Content: utils.StringValue(message.Content),
				},
			)

			if err != nil {
				return err
			}
		}

		err := s.conversationService.AddMessage(
			req.ConversationID,
			conversation.Message{
				Role:    "assistant",
				Content: assistantContent.String(),
			},
		)

		if err != nil {
			return err
		}
	}

	return nil
}

func (s *ChatService) buildMessages(
	req *chat.ChatCompletionRequest,
) ([]chat.Message, error) {

	if req.ConversationID == "" {
		return req.Messages, nil
	}

	conversation, err := s.conversationService.Get(
		req.ConversationID,
	)

	if err != nil {
		return nil, err
	}

	messages := make(
		[]chat.Message,
		0,
		len(conversation.Messages)+len(req.Messages),
	)

	for _, message := range conversation.Messages {
		messages = append(
			messages,
			chat.Message{
				Role:    message.Role,
				Content: utils.StringPtr(message.Content),
			},
		)
	}

	messages = append(
		messages,
		req.Messages...,
	)

	return messages, nil
}
