package talklikeapirate

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/generative-ai-go/genai"
	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/config"
	"google.golang.org/api/option"
)

// TalkLikeAPiratePlugin reimplements the send function
// with an AI intermediate.
type TalkLikeAPiratePlugin struct {
	client *genai.Client
	prompt string

	b bot.Bot
	c *config.Config
}

func New(c *config.Config) *TalkLikeAPiratePlugin {
	p := &TalkLikeAPiratePlugin{
		c: c,
	}

	return p
}

func (p *TalkLikeAPiratePlugin) Filter(input string) (string, error) {
	if !p.c.GetBool("talklikeapirate.enabled", false) {
		return input, nil
	}
	if p.client == nil {
		var err error
		p.client, err = p.getClient()
		if err != nil {
			return input, err
		}
	}
	model, err := p.GetModel()
	if err != nil {
		log.Error().Err(err).Send()
		return input, err
	}

	res, err := model.GenerateContent(context.Background(), genai.Text(input))
	if err != nil {
		log.Error().Err(err).Send()
		return input, err
	}

	if len(res.Candidates) == 0 {
		err := errors.New("no candidates found")
		log.Error().Err(err).Send()
		return input, err
	}

	// Need to check here that we got an actual completion, not a
	// warning about bad content. FinishReason exists on Completion.

	completion := ""
	for _, p := range res.Candidates[0].Content.Parts {
		completion += fmt.Sprintf("%s", p)
	}

	return completion, nil
}

func (p *TalkLikeAPiratePlugin) GetModel() (*genai.GenerativeModel, error) {
	model := p.client.GenerativeModel(p.c.Get("gemini.model", "gemini-1.5-flash"))
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

	if prompt := p.c.Get("talklikeapirate.systemprompt", ""); prompt != "" {
		model.SystemInstruction = &genai.Content{
			Parts: []genai.Part{genai.Text(prompt)},
		}
	} else {
		return nil, errors.New("no system prompt selected")
	}

	return model, nil
}

func (p *TalkLikeAPiratePlugin) getClient() (*genai.Client, error) {
	ctx := context.Background()
	key := p.c.Get("GEMINI_API_KEY", "")
	if key == "" {
		return nil, errors.New("missing GEMINI_API_KEY")
	}
	client, err := genai.NewClient(ctx, option.WithAPIKey(key))
	if err != nil {
		return nil, err
	}
	return client, nil
}
