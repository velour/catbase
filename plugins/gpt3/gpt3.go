package gpt3

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"regexp"

	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/config"
)

const gpt3URL = "https://api.openai.com/v1/engines/ada/completions"

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
			Regex:    regexp.MustCompile(`^gpt3 (?P<text>.*)$`),
			HelpText: "request text completion",
			Handler:  p.message,
		},
	}
	log.Debug().Msg("Registering GPT3 handlers")
	p.b.RegisterTable(p, p.h)
}

func (p *GPT3Plugin) message(r bot.Request) bool {
	stem := r.Values["text"]
	log.Debug().Msgf("Got GPT3 request: %s", stem)
	postStruct := gpt3Request{
		Prompt:      stem,
		MaxTokens:   p.c.GetInt("gpt3.tokens", 16),
		Temperature: p.c.GetFloat64("gpt3.temperature", 1),
		TopP:        p.c.GetFloat64("gpt3.top_p", 1),
		N:           p.c.GetInt("gpt3.n", 1),
		Stop:        "\n",
		Echo:        true,
	}
	postBody, _ := json.Marshal(postStruct)
	client := &http.Client{}
	req, err := http.NewRequest("POST", gpt3URL, bytes.NewBuffer(postBody))
	if err != nil {
		log.Error().Err(err).Msg("could not make gpt3 request")
		return false
	}
	gpt3Key := p.c.Get("gpt3.bearer", "")
	if gpt3Key == "" {
		log.Error().Msgf("no GPT3 key given")
		return false
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", gpt3Key))
	res, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("could not make gpt3 request")
		return false
	}

	resBody, _ := ioutil.ReadAll(res.Body)
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
	p.b.Send(r.Conn, bot.Message, r.Msg.Channel, msg)
	return true
}

type gpt3Request struct {
	Prompt      string      `json:"prompt"`
	MaxTokens   int         `json:"max_tokens"`
	Temperature float64     `json:"temperature"`
	TopP        float64     `json:"top_p"`
	N           int         `json:"n"`
	Stream      bool        `json:"stream"`
	Logprobs    interface{} `json:"logprobs"`
	Stop        string      `json:"stop"`
	Echo        bool        `json:"echo"`
}

type gpt3Response struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Text         string      `json:"text"`
		Index        int         `json:"index"`
		Logprobs     interface{} `json:"logprobs"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
}
