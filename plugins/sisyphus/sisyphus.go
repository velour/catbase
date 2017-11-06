package sisyphus

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
)

const (
	BOULDER  = ":full_moon:"
	MOUNTAIN = ":new_moon:"
)

type SisyphusPlugin struct {
	Bot       bot.Bot
	listenFor map[string]*game
}

type game struct {
	id       string
	channel  string
	bot      bot.Bot
	who      string
	start    time.Time
	size     int
	current  int
	nextPush time.Time
	nextDec  time.Time
	timers   [2]*time.Timer
	ended    bool
	nextAns  int
}

func NewRandomGame(bot bot.Bot, channel, who string) *game {
	size := rand.Intn(9) + 2
	g := game{
		channel: channel,
		bot:     bot,
		who:     who,
		start:   time.Now(),
		size:    size,
		current: size / 2,
	}
	g.id = bot.SendMessage(channel, g.toMessageString())

	g.schedulePush()
	g.scheduleDecrement()

	return &g
}

func (g *game) scheduleDecrement() {
	if g.timers[0] != nil {
		g.timers[0].Stop()
	}
	minDec := g.bot.Config().Sisyphus.MinDecrement
	maxDec := g.bot.Config().Sisyphus.MinDecrement
	g.nextDec = time.Now().Add(time.Duration((minDec + rand.Intn(maxDec))) * time.Minute)
	go func() {
		t := time.NewTimer(g.nextDec.Sub(time.Now()))
		g.timers[0] = t
		select {
		case <-t.C:
			g.handleDecrement()
		}
	}()
}

func (g *game) schedulePush() {
	if g.timers[1] != nil {
		g.timers[1].Stop()
	}
	minPush := g.bot.Config().Sisyphus.MinPush
	maxPush := g.bot.Config().Sisyphus.MaxPush
	g.nextPush = time.Now().Add(time.Duration(rand.Intn(maxPush)+minPush) * time.Minute)
	go func() {
		t := time.NewTimer(g.nextPush.Sub(time.Now()))
		g.timers[1] = t
		select {
		case <-t.C:
			g.handleNotify()
		}
	}()
}

func (g *game) endGame() {
	for _, t := range g.timers {
		t.Stop()
	}
	g.ended = true
}

func (g *game) handleDecrement() {
	g.current++
	g.bot.Edit(g.channel, g.toMessageString(), g.id)
	if g.current > g.size-2 {
		g.bot.ReplyToMessageIdentifier(g.channel, "you lose", g.id)
		msg := fmt.Sprintf("%s just lost the game after %s", g.who, time.Now().Sub(g.start))
		g.bot.SendMessage(g.channel, msg)
		g.endGame()
	} else {
		g.scheduleDecrement()
	}
}

func (g *game) handleNotify() {
	g.bot.ReplyToMessageIdentifier(g.channel, "You can push now.\n"+g.generateQuestion(), g.id)
}

func (g *game) generateQuestion() string {
	n1 := rand.Intn(99) + (rand.Intn(9)+1)*100
	n2 := rand.Intn(99) + (rand.Intn(9)+1)*100
	var op string
	switch i := rand.Intn(3); i {
	case 0:
		// times
		g.nextAns = n1 * n2
		op = "*"
	case 1:
		// plus
		g.nextAns = n1 + n2
		op = "+"
	case 2:
		// minus
		g.nextAns = n1 - n2
		op = "-"
	}
	return fmt.Sprintf("What is %d %s %d?", n1, op, n2)
}

func (g *game) checkAnswer(ans string) bool {
	if strings.Contains(ans, strconv.Itoa(g.nextAns)) {
		g.current--
		if g.current < 0 {
			g.current = 0
		}
		return true
	}
	return false
}

func (g *game) toMessageString() string {
	out := ""
	for i := 0; i < g.size; i++ {
		for j := 0; j < i; j++ {
			out = out + MOUNTAIN
		}
		if i == g.current {
			out = out + BOULDER
		} else if i == g.current+1 {
			out = out + ":" + g.who + ":"
		}
		out = out + "\n"
	}
	return out
}

func New(b bot.Bot) *SisyphusPlugin {
	return &SisyphusPlugin{
		Bot:       b,
		listenFor: map[string]*game{},
	}
}

func (p *SisyphusPlugin) Message(message msg.Message) bool {
	if strings.ToLower(message.Body) == "start sisyphus" {
		b := NewRandomGame(p.Bot, message.Channel, message.User.Name)
		p.listenFor[b.id] = b
		p.Bot.ReplyToMessageIdentifier(message.Channel, "Over here.", b.id)
		return true
	}
	return false
}

func (p *SisyphusPlugin) Help(channel string, parts []string) {
	p.Bot.SendMessage(channel, "https://en.wikipedia.org/wiki/Sisyphus")
}

func (p *SisyphusPlugin) Event(kind string, message msg.Message) bool {
	return false
}

func (p *SisyphusPlugin) BotMessage(message msg.Message) bool {
	return false
}

func (p *SisyphusPlugin) RegisterWeb() *string {
	return nil
}

func (p *SisyphusPlugin) ReplyMessage(message msg.Message, identifier string) bool {
	if strings.ToLower(message.User.Name) != strings.ToLower(p.Bot.Config().Nick) {
		if g, ok := p.listenFor[identifier]; ok {

			log.Printf("got message on %s: %+v", identifier, message)

			if g.ended {
				return false
			}

			if strings.ToLower(message.Body) == "end game" {
				g.endGame()
				return true
			}

			if time.Now().After(g.nextPush) {
				if g.checkAnswer(message.Body) {
					p.Bot.Edit(message.Channel, g.toMessageString(), identifier)
					g.schedulePush()
					msg := fmt.Sprintf("Ok. You can push again in %s", g.nextPush.Sub(time.Now()))
					p.Bot.ReplyToMessageIdentifier(message.Channel, msg, identifier)
				} else {
					p.Bot.ReplyToMessageIdentifier(message.Channel, "you lose", identifier)
					msg := fmt.Sprintf("%s just lost the sisyphus game after %s", g.who, time.Now().Sub(g.start))
					p.Bot.SendMessage(message.Channel, msg)
					g.endGame()
				}
			} else {
				p.Bot.ReplyToMessageIdentifier(message.Channel, "you cannot push yet", identifier)
			}
			return true
		}
	}
	return false
}
