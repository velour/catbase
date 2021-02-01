package gpt2

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/config"
)

type GPT2Plugin struct {
	b bot.Bot
	c *config.Config
}

func New(b bot.Bot) *GPT2Plugin {
	p := &GPT2Plugin{
		b: b,
		c: b.Config(),
	}

	b.RegisterRegexCmd(p, bot.Message, gpt2Regex, p.gpt2Cmd)
	b.Register(p, bot.Help, p.help)

	return p
}

var gpt2Regex = regexp.MustCompile(`(?i)^gpt2 (?P<input>.*)$`)

func (p *GPT2Plugin) gpt2Cmd(r bot.Request) bool {
	input := r.Values["input"]
	txt, err := p.getGPTText(input)
	if err != nil {
		txt = p.c.Get("gpt.error", "The GPT service is unavailable.")
	}
	p.b.Send(r.Conn, bot.Message, r.Msg.Channel, txt)
	return true
}

func (p *GPT2Plugin) help(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	txt := "Invoke the GPT-2 API with: `!gpt2 <your seed text>"
	p.b.Send(c, bot.Message, message.Channel, txt)
	return true
}

const separator = "<|endoftext|>"

func (p *GPT2Plugin) getGPTText(prefix string) (string, error) {
	serviceURI := p.c.Get("gpt2.service", "")

	if serviceURI == "" {
		return "", fmt.Errorf("cannot contact GPT2 service")
	}

	args := struct {
		Prefix      string  `json:"prefix"`
		Length      int     `json:"length"`
		Temperature float64 `json:"temperature"`
		TopP        float64 `json:"top_p"`
		TopK        float64 `json:"top_k"`
	}{
		Prefix:      prefix,
		Length:      p.c.GetInt("gpt2.length", 50),
		Temperature: p.c.GetFloat64("gpt2.temperature", 0.7),
		TopK:        p.c.GetFloat64("gpt2.topk", 0),
		TopP:        p.c.GetFloat64("gpt2.topp", 0),
	}

	values, _ := json.Marshal(args)

	resp, err := http.Post(serviceURI, "application/json", bytes.NewBuffer(values))
	if err != nil {
		return "", fmt.Errorf("error retrieving GPT2 response: %s", err)
	}
	if err != nil {
		return "", fmt.Errorf("error reading GPT2 response: %s", err)
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	output := struct {
		Text string `json:"text"`
	}{}
	err = dec.Decode(&output)
	if err != nil {
		return "", err
	}
	return p.cleanup(output.Text), nil
}

func (p *GPT2Plugin) cleanup(txt string) string {
	txt = strings.Split(txt, separator)[0]
	if !strings.HasSuffix(txt, ".") && strings.Count(txt, ".") > 1 {
		idx := strings.LastIndexAny(txt, ".")
		txt = txt[:idx+1]
	}
	txt = strings.TrimSpace(txt)
	return txt
}
