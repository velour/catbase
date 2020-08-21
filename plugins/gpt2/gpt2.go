package gpt2

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
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

	b.Register(p, bot.Message, p.message)
	b.Register(p, bot.Help, p.help)

	return p
}

const prefix = "gpt2"

func (p *GPT2Plugin) message(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	ch := message.Channel
	lowerBody := strings.ToLower(message.Body)
	if message.Command && strings.HasPrefix(lowerBody, prefix) {
		input := message.Body[len(prefix)+1:]
		p.b.Send(c, bot.Message, ch, p.getGPTText(input))
		return true
	}
	return false
}

func (p *GPT2Plugin) help(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	txt := "Invoke the GPT-2 API with: `!gpt2 <your seed text>"
	p.b.Send(c, bot.Message, message.Channel, txt)
	return true
}

const separator = "<|endoftext|>"

func (p *GPT2Plugin) getGPTText(input string) string {
	serviceURI := p.c.Get("gpt.service", "")
	if serviceURI == "" {
		return "Cannot contact GPT2 service."
	}
	values := url.Values{}
	values.Add("text", input)
	resp, err := http.PostForm(serviceURI, values)
	if err != nil {
		return fmt.Sprintf("Error retrieving GPT2 response: %s", err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("Error reading GPT2 response: %s", err)
	}
	resp.Body.Close()
	txt := p.cleanup(input + string(body))
	return txt
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
