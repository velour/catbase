package sisyphus

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
)

const (
	BOULDER  = ":full_moon:"
	MOUNTAIN = ":new_moon:"
)

type SisyphusPlugin struct {
	bot       bot.Bot
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

func NewRandomGame(c bot.Connector, b bot.Bot, channel, who string) *game {
	size := rand.Intn(9) + 2
	g := game{
		channel: channel,
		bot:     b,
		who:     who,
		start:   time.Now(),
		size:    size,
		current: size / 2,
	}
	g.id, _ = b.Send(c, bot.Message, channel, g.toMessageString())

	g.schedulePush(c)
	g.scheduleDecrement(c)

	return &g
}

func (g *game) scheduleDecrement(c bot.Connector) {
	if g.timers[0] != nil {
		g.timers[0].Stop()
	}
	minDec := g.bot.Config().GetInt("Sisyphus.MinDecrement", 10)
	maxDec := g.bot.Config().GetInt("Sisyphus.MaxDecrement", 30)
	g.nextDec = time.Now().Add(time.Duration(minDec+rand.Intn(maxDec)) * time.Minute)
	go func() {
		t := time.NewTimer(g.nextDec.Sub(time.Now()))
		g.timers[0] = t
		select {
		case <-t.C:
			g.handleDecrement(c)
		}
	}()
}

func (g *game) schedulePush(c bot.Connector) {
	if g.timers[1] != nil {
		g.timers[1].Stop()
	}
	minPush := g.bot.Config().GetInt("Sisyphus.MinPush", 1)
	maxPush := g.bot.Config().GetInt("Sisyphus.MaxPush", 10)
	g.nextPush = time.Now().Add(time.Duration(rand.Intn(maxPush)+minPush) * time.Minute)
	go func() {
		t := time.NewTimer(g.nextPush.Sub(time.Now()))
		g.timers[1] = t
		select {
		case <-t.C:
			g.handleNotify(c)
		}
	}()
}

func (g *game) endGame() {
	for _, t := range g.timers {
		t.Stop()
	}
	g.ended = true
}

func (g *game) handleDecrement(c bot.Connector) {
	g.current++
	g.bot.Send(c, bot.Edit, g.channel, g.toMessageString(), g.id)
	if g.current > g.size-2 {
		g.bot.Send(c, bot.Reply, g.channel, "you lose", g.id)
		msg := fmt.Sprintf("%s just lost the game after %s", g.who, time.Now().Sub(g.start))
		g.bot.Send(c, bot.Message, g.channel, msg)
		g.endGame()
	} else {
		g.scheduleDecrement(c)
	}
}

func (g *game) handleNotify(c bot.Connector) {
	g.bot.Send(c, bot.Reply, g.channel, "You can push now.\n"+g.generateQuestion(), g.id)
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
	sp := &SisyphusPlugin{
		bot:       b,
		listenFor: map[string]*game{},
	}
	b.Register(sp, bot.Message, sp.message)
	b.Register(sp, bot.Reply, sp.replyMessage)
	b.Register(sp, bot.Help, sp.help)
	return sp
}

func (p *SisyphusPlugin) message(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	if strings.ToLower(message.Body) == "start sisyphus" {
		b := NewRandomGame(c, p.bot, message.Channel, message.User.Name)
		p.listenFor[b.id] = b
		p.bot.Send(c, bot.Reply, message.Channel, "Over here.", b.id)
		return true
	}
	return false
}

func (p *SisyphusPlugin) help(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	p.bot.Send(c, bot.Message, message.Channel, "https://en.wikipedia.org/wiki/Sisyphus")
	return true
}

func (p *SisyphusPlugin) replyMessage(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	identifier := args[0].(string)
	if strings.ToLower(message.User.Name) != strings.ToLower(p.bot.Config().Get("Nick", "bot")) {
		if g, ok := p.listenFor[identifier]; ok {

			log.Debug().Msgf("got message on %s: %+v", identifier, message)

			if g.ended {
				return false
			}

			if strings.ToLower(message.Body) == "end game" {
				g.endGame()
				return true
			}

			if time.Now().After(g.nextPush) {
				if g.checkAnswer(message.Body) {
					p.bot.Send(c, bot.Edit, message.Channel, g.toMessageString(), identifier)
					g.schedulePush(c)
					msg := fmt.Sprintf("Ok. You can push again in %s", g.nextPush.Sub(time.Now()))
					p.bot.Send(c, bot.Reply, message.Channel, msg, identifier)
				} else {
					p.bot.Send(c, bot.Reply, message.Channel, "you lose", identifier)
					msg := fmt.Sprintf("%s just lost the sisyphus game after %s", g.who, time.Now().Sub(g.start))
					p.bot.Send(c, bot.Message, message.Channel, msg)
					g.endGame()
				}
			} else {
				p.bot.Send(c, bot.Reply, message.Channel, "you cannot push yet", identifier)
			}
			return true
		}
	}
	return false
}
