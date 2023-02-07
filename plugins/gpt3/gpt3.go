package gpt3

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"reflect"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/config"
)

const gpt3URL = "https://api.openai.com/v1/engines/%s/completions"
const gpt3ModURL = "https://api.openai.com/v1/moderations"

type GPT3Plugin struct {
	b bot.Bot
	c *config.Config
	h bot.HandlerTable
}

func New(b bot.Bot) *GPT3Plugin {
	p := &GPT3Plugin{
		b: b,
		c: b.Config(),
	}
	p.register()
	return p
}

func (p *GPT3Plugin) register() {
	p.h = bot.HandlerTable{
		{
			Kind: bot.Message, IsCmd: true,
			Regex:    regexp.MustCompile(`(?is)^gpt3 (?P<text>.*)`),
			HelpText: "request text completion",
			Handler:  p.message,
		},
	}
	log.Debug().Msg("Registering GPT3 handlers")
	p.b.RegisterTable(p, p.h)
}

func (p *GPT3Plugin) message(r bot.Request) bool {
	stem := r.Values["text"]
	p.b.Send(r.Conn, bot.Message, r.Msg.Channel, p.mkRequest(stem))
	return true
}

func (p *GPT3Plugin) mkRequest(stem string) string {
	log.Debug().Msgf("Got GPT3 request: %s", stem)
	if err := p.checkStem(stem); err != nil {
		return "GPT3 Moderation " + err.Error()
	}
	postStruct := gpt3Request{
		Prompt:      stem,
		MaxTokens:   p.c.GetInt("gpt3.tokens", 16),
		Temperature: p.c.GetFloat64("gpt3.temperature", 1),
		TopP:        p.c.GetFloat64("gpt3.top_p", 1),
		N:           p.c.GetInt("gpt3.n", 1),
		Stop:        p.c.GetArray("gpt3.stop", []string{"\n"}),
		Echo:        true,
	}
	postBody, _ := json.Marshal(postStruct)
	client := &http.Client{}
	u := fmt.Sprintf(gpt3URL, p.c.Get("gpt3.engine", "ada"))
	req, err := http.NewRequest("POST", u, bytes.NewBuffer(postBody))
	if err != nil {
		log.Error().Err(err).Msg("could not make gpt3 request")
		return err.Error()
	}
	gpt3Key := p.c.Get("gpt3.bearer", "")
	if gpt3Key == "" {
		log.Error().Msgf("no GPT3 key given")
		return "No GPT3 API key"
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", gpt3Key))
	res, err := client.Do(req)
	if err != nil {
		return err.Error()
	}

	resBody, _ := io.ReadAll(res.Body)
	gpt3Resp := gpt3Response{}
	err = json.Unmarshal(resBody, &gpt3Resp)

	log.Debug().
		Str("body", string(resBody)).
		Interface("resp", gpt3Resp).
		Msg("OpenAI Response")

	msg := "OpenAI is too shitty to respond to that."
	if len(gpt3Resp.Choices) > 0 {
		msg = gpt3Resp.Choices[rand.Intn(len(gpt3Resp.Choices))].Text
	}
	return msg
}

func (p *GPT3Plugin) checkStem(stem string) error {
	postBody, _ := json.Marshal(gpt3ModRequest{Input: stem})
	client := &http.Client{}
	req, err := http.NewRequest("POST", gpt3ModURL, bytes.NewBuffer(postBody))
	if err != nil {
		return err
	}
	gpt3Key := p.c.Get("gpt3.bearer", "")
	if gpt3Key == "" {
		return fmt.Errorf("no GPT3 API key")
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", gpt3Key))
	res, err := client.Do(req)
	if err != nil {
		return err
	}

	resBody, _ := io.ReadAll(res.Body)
	log.Debug().Str("resBody", string(resBody)).Msg("res")
	gpt3Resp := gpt3Moderation{}
	err = json.Unmarshal(resBody, &gpt3Resp)
	if err != nil {
		return err
	}
	log.Debug().Interface("GPT3 Moderation", gpt3Resp).Msg("Moderation result")
	for _, res := range gpt3Resp.Results {
		if res.Flagged {
			list := ""
			categories := reflect.ValueOf(res.Categories)
			fields := reflect.VisibleFields(reflect.TypeOf(res.Categories))
			for i := 0; i < categories.NumField(); i++ {
				if categories.Field(i).Bool() {
					list += fields[i].Name + ", "
				}
			}
			list = strings.TrimSuffix(list, ", ")
			return fmt.Errorf("flagged: %s", list)
		}
	}
	return nil
}

type gpt3Request struct {
	Prompt      string   `json:"prompt"`
	MaxTokens   int      `json:"max_tokens"`
	Temperature float64  `json:"temperature"`
	TopP        float64  `json:"top_p"`
	N           int      `json:"n"`
	Stream      bool     `json:"stream"`
	Logprobs    any      `json:"logprobs"`
	Stop        []string `json:"stop"`
	Echo        bool     `json:"echo"`
}

type gpt3ModRequest struct {
	Input string `json:"input"`
}

type gpt3Response struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Text         string `json:"text"`
		Index        int    `json:"index"`
		Logprobs     any    `json:"logprobs"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

type gpt3Moderation struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Results []struct {
		Categories struct {
			Hate            bool `json:"hate"`
			HateThreatening bool `json:"hate/threatening"`
			SelfHarm        bool `json:"self-harm"`
			Sexual          bool `json:"sexual"`
			SexualMinors    bool `json:"sexual/minors"`
			Violence        bool `json:"violence"`
			ViolenceGraphic bool `json:"violence/graphic"`
		} `json:"categories"`
		CategoryScores struct {
			Hate            float64 `json:"hate"`
			HateThreatening float64 `json:"hate/threatening"`
			SelfHarm        float64 `json:"self-harm"`
			Sexual          float64 `json:"sexual"`
			SexualMinors    float64 `json:"sexual/minors"`
			Violence        float64 `json:"violence"`
			ViolenceGraphic float64 `json:"violence/graphic"`
		} `json:"category_scores"`
		Flagged bool `json:"flagged"`
	} `json:"results"`
}
