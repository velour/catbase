package tldr

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/google/generative-ai-go/genai"
	"github.com/velour/catbase/config"
	"google.golang.org/api/option"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"

	"github.com/rs/zerolog/log"
)

const templateKey = "tldr.prompttemplate"

var defaultTemplate = "Summarize the following conversation:\n"

type TLDRPlugin struct {
	b           bot.Bot
	c           *config.Config
	history     map[string][]history
	index       int
	lastRequest time.Time
}

type history struct {
	timestamp time.Time
	user      string
	body      string
}

func New(b bot.Bot) *TLDRPlugin {
	plugin := &TLDRPlugin{
		b:           b,
		c:           b.Config(),
		history:     map[string][]history{},
		index:       0,
		lastRequest: time.Now().Add(-24 * time.Hour),
	}
	plugin.register()
	return plugin
}

func (p *TLDRPlugin) register() {
	p.b.RegisterTable(p, bot.HandlerTable{
		{
			Kind: bot.Message, IsCmd: true,
			Regex:    regexp.MustCompile(`tl;?dr-prompt$`),
			HelpText: "Get the tl;dr prompt",
			Handler:  p.squawkTLDR,
		},
		{
			Kind: bot.Message, IsCmd: true,
			Regex:    regexp.MustCompile(`tl;?dr-prompt reset`),
			HelpText: "Reset the tl;dr prompt",
			Handler:  p.resetTLDR,
		},
		{
			Kind: bot.Message, IsCmd: true,
			Regex:    regexp.MustCompile(`tl;?dr-prompt (?P<prompt>.*)`),
			HelpText: "Set the tl;dr prompt",
			Handler:  p.setTLDR,
		},
		{
			Kind: bot.Message, IsCmd: true,
			Regex:    regexp.MustCompile(`tl;?dr`),
			HelpText: "Get a summary of the channel",
			Handler:  p.betterTLDR,
		},
		{
			Kind: bot.Message, IsCmd: false,
			Regex:   regexp.MustCompile(`.*`),
			Handler: p.record,
		},
	})
	p.b.Register(p, bot.Help, p.help)
}

func (p *TLDRPlugin) record(r bot.Request) bool {
	hist := history{
		body:      strings.ToLower(r.Msg.Body),
		user:      r.Msg.User.Name,
		timestamp: time.Now(),
	}
	p.addHistory(r.Msg.Channel, hist)

	return false
}

func (p *TLDRPlugin) addHistory(ch string, hist history) {
	p.history[ch] = append(p.history[ch], hist)
	sz := len(p.history[ch])
	max := p.b.Config().GetInt("TLDR.HistorySize", 1000)
	keepHrs := time.Duration(p.b.Config().GetInt("TLDR.KeepHours", 24))
	// Clamp the size of the history
	if sz > max {
		p.history[ch] = p.history[ch][len(p.history)-max:]
	}
	// Remove old entries
	yesterday := time.Now().Add(-keepHrs * time.Hour)
	begin := 0
	for i, m := range p.history[ch] {
		if !m.timestamp.Before(yesterday) {
			begin = i - 1 // should keep this message
			if begin < 0 {
				begin = 0
			}
			break
		}
	}
	p.history[ch] = p.history[ch][begin:]
}

// Help responds to help requests. Every plugin must implement a help function.
func (p *TLDRPlugin) help(c bot.Connector, kind bot.Kind, message msg.Message, args ...any) bool {
	p.b.Send(c, bot.Message, message.Channel, "tl;dr")
	return true
}

func (p *TLDRPlugin) betterTLDR(r bot.Request) bool {
	ch := r.Msg.Channel
	c, err := p.getClient()
	if err != nil {
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "Couldn't fetch an AI client")
		return true
	}
	promptConfig := p.c.Get(templateKey, defaultTemplate)
	promptTpl := template.Must(template.New("gptprompt").Parse(promptConfig))
	prompt := bytes.Buffer{}
	data := p.c.GetMap("tldr.promptdata", map[string]string{})
	promptTpl.Execute(&prompt, data)
	backlog := ""
	maxLen := p.c.GetInt("tldr.maxgpt", 4096)
	for i := len(p.history[ch]) - 1; i >= 0; i-- {
		h := p.history[ch][i]
		str := fmt.Sprintf("%s: %s\n", h.user, h.body)
		if len(backlog) > maxLen {
			break
		}
		backlog = str + backlog
	}

	model := c.GenerativeModel(p.c.Get("gemini.model", "gemini-1.5-flash"))
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(prompt.String())},
	}
	res, err := model.GenerateContent(context.Background(), genai.Text(backlog))
	if err != nil {
		log.Error().Err(err).Send()
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "Error: could not generate a TLDR")
		return true
	}
	log.Debug().
		Str("prompt", prompt.String()).
		Str("backlog", backlog).
		Interface("completion", res.Candidates).
		Msgf("tl;dr")

	if len(res.Candidates) == 0 {
		log.Error().Err(errors.New("no candidates found")).Send()
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "Error: no candidates generating a TLDR")
		return true
	}

	completion := ""
	for _, p := range res.Candidates[0].Content.Parts {
		completion += fmt.Sprintf("%s", p)
	}

	p.b.Send(r.Conn, bot.Message, r.Msg.Channel, completion)
	return true
}

func (p *TLDRPlugin) squawkTLDR(r bot.Request) bool {
	prompt := p.c.Get(templateKey, defaultTemplate)
	p.b.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf(`Current prompt is: "%s"`,
		strings.TrimSpace(prompt)))
	return true
}

func (p *TLDRPlugin) resetTLDR(r bot.Request) bool {
	p.c.Set(templateKey, defaultTemplate)
	p.b.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf(`Set prompt to: "%s"`,
		strings.TrimSpace(defaultTemplate)))
	return true
}

func (p *TLDRPlugin) setTLDR(r bot.Request) bool {
	prompt := r.Values["prompt"] + "\n"
	p.c.Set(templateKey, prompt)
	p.b.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf(`Set prompt to: "%s"`, strings.TrimSpace(prompt)))
	return true
}

func (p *TLDRPlugin) getClient() (*genai.Client, error) {
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
