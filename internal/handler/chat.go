package handler

import (
	"encoding/json"
	"net/http"

	"ai_gateway/internal/chat"
	"ai_gateway/internal/errors"
	"ai_gateway/internal/service"
)

type ChatHandler struct {
	chatService *service.ChatService
}

func NewChatHandler(chatService *service.ChatService) *ChatHandler {
	return &ChatHandler{
		chatService: chatService,
	}
}

func (h *ChatHandler) ChatCompletions(
	w http.ResponseWriter,
	r *http.Request,
) {

	if r.Method != http.MethodPost {
		WriteError(w, &errors.Error{
			HTTPStatus: http.StatusMethodNotAllowed,
			Message:    "Method Not Allowed",
			Type:       errors.CodeInvalidRequest,
			Code:       "method_not_allowed",
		})
		return
	}

	var req chat.ChatCompletionRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, &errors.Error{
			HTTPStatus: http.StatusBadRequest,
			Message:    "Invalid JSON",
			Type:       errors.CodeInvalidRequest,
			Code:       "invalid_json",
		})
		return
	}

	if req.Model == "" {
		WriteError(w, &errors.Error{
			HTTPStatus: http.StatusBadRequest,
			Message:    "model is required",
			Type:       errors.CodeInvalidRequest,
			Code:       "missing_model",
		})
		return
	}

	if len(req.Messages) == 0 {
		WriteError(w, &errors.Error{
			HTTPStatus: http.StatusBadRequest,
			Message:    "messages is required",
			Type:       errors.CodeInvalidRequest,
			Code:       "missing_messages",
		})
		return
	}

	if req.Stream {
		h.handleStream(
			w,
			r,
			&req,
		)
		return
	}

	response, err := h.chatService.Chat(
		r.Context(),
		&req,
	)

	if err != nil {
		WriteError(w, err)
		return
	}

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		return
	}
}

func writeSSE(
	w http.ResponseWriter,
	chunk *chat.ChatCompletionChunk,
) error {

	data, err := json.Marshal(chunk)
	if err != nil {
		return err
	}

	_, err = w.Write(
		[]byte("data: " + string(data) + "\n\n"),
	)

	if err != nil {
		return err
	}

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	return nil
}

func (h *ChatHandler) handleStream(
	w http.ResponseWriter,
	r *http.Request,
	req *chat.ChatCompletionRequest,
) {

	w.Header().Set(
		"Content-Type",
		"text/event-stream",
	)

	w.Header().Set(
		"Cache-Control",
		"no-cache",
	)

	w.Header().Set(
		"Connection",
		"keep-alive",
	)

	err := h.chatService.ChatStream(
		r.Context(),
		req,
		func(chunk *chat.ChatCompletionChunk) error {
			return writeSSE(w, chunk)
		},
	)

	if err != nil {
		// 流已经开始后，不能再正常修改 HTTP Status Code。
		return
	}

	_, _ = w.Write(
		[]byte("data: [DONE]\n\n"),
	)

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}
