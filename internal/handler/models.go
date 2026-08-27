package handler

import (
	"encoding/json"
	"net/http"

	"ai_gateway/internal/model"
)

type ModelsResponse struct {
	Object string        `json:"object"`
	Data   []model.Model `json:"data"`
}

func Models(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	response := ModelsResponse{
		Object: "list",
		Data: []model.Model{
			{
				ID:      "qwen3",
				Object:  "model",
				OwnedBy: "company",
			},
			{
				ID:      "deepseek",
				Object:  "model",
				OwnedBy: "company",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
