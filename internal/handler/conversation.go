package handler

import (
	"encoding/json"
	"net/http"

	"ai_gateway/internal/errors"
	"ai_gateway/internal/service"
)

type ConversationHandler struct {
	conversationService *service.ConversationService
}

func NewConversationHandler(
	conversationService *service.ConversationService,
) *ConversationHandler {
	return &ConversationHandler{
		conversationService: conversationService,
	}
}

type createConversationRequest struct {
	Model string `json:"model"`
}

type createConversationResponse struct {
	ID     string `json:"id"`
	Object string `json:"object"`
	Model  string `json:"model"`
}

func (h *ConversationHandler) Create(
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

	var req createConversationRequest

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

	conversation, err := h.conversationService.Create(
		req.Model,
	)

	if err != nil {
		WriteError(w, err)
		return
	}

	response := createConversationResponse{
		ID:     conversation.ID,
		Object: "conversation",
		Model:  conversation.Model,
	}

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		return
	}
}
