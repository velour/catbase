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
var empty = llamaResponse{}

func (g *LLMPlugin) llama() (chatEntry, error) {
	llamaURL := g.c.GetArray("gpt.llamaurls", []string{})
	if len(llamaURL) == 0 {
		return chatEntry{}, fmt.Errorf("could not find llama url")
	}
	llamaModel := g.c.Get("gpt.llamamodel", "")
	if llamaModel == "" {
		return chatEntry{}, fmt.Errorf("could not find llama model")
	}

	prompt := g.c.Get("gpt.lastprompt", g.c.Get("gpt.prompt", ""))
	hist := []chatEntry{{
		Role:    "system",
		Content: prompt,
	}}
	hist = append(hist, g.chatHistory...)

	req := llamaRequest{
		Model:    llamaModel,
		Messages: hist,
		Stream:   false,
	}

	for _, u := range llamaURL {
		if err := g.healthCheck(u); err != nil {
			continue
		}
		llamaResp, err := mkRequest(u, req)
		if err != nil {
			continue
		}

		return llamaResp.Message, nil
	}

	return chatEntry{}, InstanceNotFoundError
}

func (p *LLMPlugin) healthCheck(llamaURL string) error {
	timeout := p.c.GetInt("gpt.timeout", 1000)
	req, _ := http.NewRequest("get", llamaURL, nil)
	client := http.Client{
		Timeout: time.Duration(timeout) * time.Millisecond,
	}
	_, err := client.Do(req)
	if err != nil {
		return err
	}
	return nil
}

func mkRequest(llamaURL string, req llamaRequest) (llamaResponse, error) {
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

	return llamaResp, nil
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
