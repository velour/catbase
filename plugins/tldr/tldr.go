package tldr

import (
	"bytes"
	"context"
	"fmt"
	"github.com/andrewstuart/openai"
	"github.com/velour/catbase/config"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"

	"github.com/rs/zerolog/log"

	"github.com/james-bowman/nlp"
)

type TLDRPlugin struct {
	b           bot.Bot
	c           *config.Config
	history     []history
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
		history:     []history{},
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
			Regex:    regexp.MustCompile(`old tl;dr`),
			HelpText: "Get a rather inaccurate summary of the channel",
			Handler:  p.tldrCmd,
		},
		{
			Kind: bot.Message, IsCmd: true,
			Regex:    regexp.MustCompile(`tl;dr`),
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

func (p *TLDRPlugin) tldrCmd(r bot.Request) bool {
	timeLimit := time.Duration(p.b.Config().GetInt("TLDR.HourLimit", 1))
	if p.lastRequest.After(time.Now().Add(-timeLimit * time.Hour)) {
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "Slow down, cowboy. Read that tiny backlog.")
		return true
	}
	return false
}

func (p *TLDRPlugin) record(r bot.Request) bool {
	hist := history{
		body:      strings.ToLower(r.Msg.Body),
		user:      r.Msg.User.Name,
		timestamp: time.Now(),
	}
	p.addHistory(hist)

	return false
}

func (p *TLDRPlugin) oldTLDR(r bot.Request) bool {
	p.lastRequest = time.Now()
	nTopics := p.b.Config().GetInt("TLDR.Topics", 5)

	stopWordSlice := p.b.Config().GetArray("TLDR.StopWords", []string{})
	if len(stopWordSlice) == 0 {
		stopWordSlice = THESE_ARE_NOT_THE_WORDS_YOU_ARE_LOOKING_FOR
		p.b.Config().SetArray("TLDR.StopWords", stopWordSlice)
	}

	vectoriser := nlp.NewCountVectoriser(stopWordSlice...)
	lda := nlp.NewLatentDirichletAllocation(nTopics)
	pipeline := nlp.NewPipeline(vectoriser, lda)
	docsOverTopics, err := pipeline.FitTransform(p.getTopics()...)

	if err != nil {
		log.Error().Err(err)
		return false
	}

	bestScores := make([][]float64, nTopics)
	bestDocs := make([][]history, nTopics)

	supportingDocs := p.b.Config().GetInt("TLDR.Support", 3)
	for i := 0; i < nTopics; i++ {
		bestScores[i] = make([]float64, supportingDocs)
		bestDocs[i] = make([]history, supportingDocs)
	}

	dr, dc := docsOverTopics.Dims()
	for topic := 0; topic < dr; topic++ {
		minScore, minIndex := min(bestScores[topic])

		for doc := 0; doc < dc; doc++ {
			score := docsOverTopics.At(topic, doc)
			if score > minScore {
				bestScores[topic][minIndex] = score
				bestDocs[topic][minIndex] = p.history[doc]
				minScore, minIndex = min(bestScores[topic])
			}
		}
	}

	topicsOverWords := lda.Components()
	tr, tc := topicsOverWords.Dims()

	vocab := make([]string, len(vectoriser.Vocabulary))
	for k, v := range vectoriser.Vocabulary {
		vocab[v] = k
	}

	response := "Here you go captain 'too good to read backlog':\n"

	for topic := 0; topic < tr; topic++ {
		bestScore := -1.
		bestTopic := ""
		for word := 0; word < tc; word++ {
			score := topicsOverWords.At(topic, word)
			if score > bestScore {
				bestScore = score
				bestTopic = vocab[word]
			}
		}
		response += fmt.Sprintf("\n*Topic #%d: %s*\n", topic, bestTopic)
		for i := range bestDocs[topic] {
			response += fmt.Sprintf("<%s>%s\n", bestDocs[topic][i].user, bestDocs[topic][i].body)
		}

	}

	p.b.Send(r.Conn, bot.Message, r.Msg.Channel, response)

	return true
}

func (p *TLDRPlugin) addHistory(hist history) {
	p.history = append(p.history, hist)
	sz := len(p.history)
	max := p.b.Config().GetInt("TLDR.HistorySize", 1000)
	keepHrs := time.Duration(p.b.Config().GetInt("TLDR.KeepHours", 24))
	// Clamp the size of the history
	if sz > max {
		p.history = p.history[len(p.history)-max:]
	}
	// Remove old entries
	yesterday := time.Now().Add(-keepHrs * time.Hour)
	begin := 0
	for i, m := range p.history {
		if !m.timestamp.Before(yesterday) {
			begin = i - 1 // should keep this message
			if begin < 0 {
				begin = 0
			}
			break
		}
	}
	p.history = p.history[begin:]
}

func (p *TLDRPlugin) getTopics() []string {
	hist := []string{}
	for _, h := range p.history {
		hist = append(hist, h.body)
	}
	return hist
}

// Help responds to help requests. Every plugin must implement a help function.
func (p *TLDRPlugin) help(c bot.Connector, kind bot.Kind, message msg.Message, args ...any) bool {
	p.b.Send(c, bot.Message, message.Channel, "tl;dr")
	return true
}

func min(slice []float64) (float64, int) {
	minVal := 1.
	minIndex := -1
	for index, val := range slice {
		if val < minVal {
			minVal = val
			minIndex = index
		}
	}
	return minVal, minIndex
}

func (p *TLDRPlugin) betterTLDR(r bot.Request) bool {
	c, err := p.getClient()
	if err != nil {
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "Couldn't fetch an OpenAI client")
		return true
	}
	promptConfig := p.c.Get("tldr.prompttemplate", "Summarize the following conversation:\n")
	promptTpl := template.Must(template.New("gptprompt").Parse(promptConfig))
	prompt := bytes.Buffer{}
	data := p.c.GetMap("tldr.promptdata", map[string]string{})
	promptTpl.Execute(&prompt, data)
	backlog := ""
	for _, h := range p.history {
		backlog += fmt.Sprintf("%s: %s\n", h.user, h.body)
	}
	sess := c.NewChatSession(prompt.String())
	completion, err := sess.Complete(context.TODO(), backlog)
	if err != nil {
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "Couldn't run the OpenAI request")
		return true
	}
	log.Debug().
		Str("prompt", prompt.String()).
		Str("backlog", backlog).
		Str("completion", completion).
		Msgf("tl;dr")
	p.b.Send(r.Conn, bot.Message, r.Msg.Channel, completion)
	return true
}

func (p *TLDRPlugin) getClient() (*openai.Client, error) {
	token := p.c.Get("gpt.token", "")
	if token == "" {
		return nil, fmt.Errorf("no GPT token given")
	}
	return openai.NewClient(token)
}
