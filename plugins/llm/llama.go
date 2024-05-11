package llm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

var InstanceNotFoundError = errors.New("instance not found")
var empty = chatEntry{}

func (g *LLMPlugin) llama() (chatEntry, error) {
	llamaURL := g.c.Get("gpt.llamaurl", "")
	if llamaURL == "" {
		return empty, fmt.Errorf("could not find llama url")
	}
	llamaModel := g.c.Get("gpt.llamamodel", "")
	if llamaModel == "" {
		return empty, fmt.Errorf("could not find llama model")
	}

	req := llamaRequest{
		Model:    llamaModel,
		Messages: g.chatHistory,
		Stream:   false,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return empty, fmt.Errorf("could not marshal llama request: %w", err)
	}

	resp, err := http.Post(llamaURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return empty, fmt.Errorf("could not post llama request: %w", err)
	}

	if resp.StatusCode == 503 {
		return empty, InstanceNotFoundError
	}
	body, _ = io.ReadAll(resp.Body)

	llamaResp := llamaResponse{}
	err = json.Unmarshal(body, &llamaResp)
	if err != nil {
		return empty, fmt.Errorf("could not unmarshal llama response: %w, raw: %s", err, string(body))
	}

	return llamaResp.Message, nil
}

type llamaRequest struct {
	Model    string      `json:"model"`
	Stream   bool        `json:"stream"`
	Messages []chatEntry `json:"messages"`
}

type llamaResponse struct {
	Model              string    `json:"model"`
	CreatedAt          time.Time `json:"created_at"`
	Message            chatEntry `json:"message"`
	DoneReason         string    `json:"done_reason"`
	Done               bool      `json:"done"`
	TotalDuration      int64     `json:"total_duration"`
	LoadDuration       int       `json:"load_duration"`
	PromptEvalDuration int       `json:"prompt_eval_duration"`
	EvalCount          int       `json:"eval_count"`
	EvalDuration       int64     `json:"eval_duration"`
}
