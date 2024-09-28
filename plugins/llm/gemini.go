package llm

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/generative-ai-go/genai"
	"github.com/rs/zerolog/log"
	"google.golang.org/api/option"
	"strings"
)

func (p *LLMPlugin) geminiConnect() error {
	ctx := context.Background()
	key := p.c.Get("GEMINI_API_KEY", "")
	if key == "" {
		return errors.New("missing GEMINI_API_KEY")
	}
	client, err := genai.NewClient(ctx, option.WithAPIKey(key))
	if err != nil {
		return err
	}
	p.geminiClient = client
	return nil
}

func (p *LLMPlugin) gemini(msg string) (chatEntry, error) {
	model := p.geminiClient.GenerativeModel(p.c.Get("gemini.model", "gemini-1.5-flash"))
	model.SetMaxOutputTokens(int32(p.c.GetInt("gemini.maxtokens", 100)))
	model.SetTopP(float32(p.c.GetFloat64("gemini.topp", 0.95)))
	model.SetTopK(int32(p.c.GetInt("gemini.topk", 20)))
	model.SetTemperature(float32(p.c.GetFloat64("gemini.temp", 0.9)))

	model.SafetySettings = []*genai.SafetySetting{
		{genai.HarmCategoryHarassment, genai.HarmBlockNone},
		{genai.HarmCategoryHateSpeech, genai.HarmBlockNone},
		{genai.HarmCategorySexuallyExplicit, genai.HarmBlockNone},
		{genai.HarmCategoryDangerousContent, genai.HarmBlockNone},
	}

	if prompt := p.c.Get("gemini.systemprompt", ""); prompt != "" {
		model.SystemInstruction = &genai.Content{
			Parts: []genai.Part{genai.Text(prompt)},
		}
	}

	cs := model.StartChat()

	ctx := context.Background()

	cs.History = []*genai.Content{}
	for _, h := range p.getChatHistory() {
		cs.History = append(cs.History, &genai.Content{
			Parts: []genai.Part{
				genai.Text(h.Content),
			},
			Role: h.Role,
		})
	}

	res, err := cs.SendMessage(ctx, genai.Text(msg))
	if err != nil {
		log.Error().Err(err).Send()
		return chatEntry{}, err
	}
	if len(res.Candidates) == 0 {
		return chatEntry{}, errors.New("no candidates")
	}
	c := res.Candidates[0]
	output := ""
	for _, p := range c.Content.Parts {
		output = fmt.Sprintf("%s %s", output, p)
	}
	return chatEntry{
		Role:    c.Content.Role,
		Content: strings.TrimSpace(output),
	}, nil
}
