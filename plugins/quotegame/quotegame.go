package quotegame

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/config"
)

type QuoteGame struct {
	b  bot.Bot
	c  *config.Config
	db *sqlx.DB

	handlers []bot.HandlerSpec

	currentGame *time.Timer
	currentName string
}

func New(b bot.Bot) *QuoteGame {
	p := &QuoteGame{
		b:           b,
		c:           b.Config(),
		db:          b.DB(),
		currentGame: nil,
	}
	p.register()
	return p
}

func (p *QuoteGame) register() {
	log.Debug().Msg("registering quote handlers")
	p.handlers = []bot.HandlerSpec{
		{
			Kind: bot.Message, IsCmd: false,
			Regex:    regexp.MustCompile(`(?i)^quote game$`),
			HelpText: "Start a quote game",
			Handler:  p.startGame,
		},
		{
			Kind: bot.Message, IsCmd: false,
			Regex:   regexp.MustCompile(`(?i)^guess:\s?(?P<name>.+)$`),
			Handler: p.guess,
		},
	}
	p.b.RegisterTable(p, p.handlers)
}

type quote struct {
	Fact   string
	Tidbit string
}

func (p *QuoteGame) getAllQuotes() ([]quote, error) {
	threshold := p.c.GetInt("quotegame.threshold", 10)
	q := `select fact, tidbit from factoid where fact like '%quotes' group by fact having count(fact) > ?`
	quotes := []quote{}
	err := p.db.Select(&quotes, q, threshold)
	if err != nil {
		return nil, err
	}
	return quotes, nil
}

func (p *QuoteGame) getRandomquote() (string, string, error) {
	quotes, err := p.getAllQuotes()
	if err != nil {
		return "", "", err
	}

	quote := quotes[rand.Intn(len(quotes))]
	who := strings.ReplaceAll(quote.Fact, " quotes", "")
	what := strings.ReplaceAll(quote.Tidbit, who, "")
	what = strings.Trim(what, " <>")

	return who, what, nil
}

func (p *QuoteGame) startGame(r bot.Request) bool {
	log.Debug().Msg("startGame called")
	if p.currentGame != nil {
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "There is already a quote game running.")
		return true
	}

	who, quote, err := p.getRandomquote()
	if err != nil {
		log.Error().Err(err).Msg("problem getting quote")
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "Error: "+err.Error())
		return true
	}

	length := time.Duration(p.c.GetInt("quotegame.length", 120))
	p.currentGame = time.AfterFunc(length*time.Second, func() {
		p.currentGame = nil
		p.currentName = ""
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel,
			fmt.Sprintf("The quote game ended and nobody won. The answer was %s", who))
	})

	p.currentName = who

	msg := fmt.Sprintf("Quote game: Who said \"%s\"?\nYou have %d seconds to guess.\nUse `guess: name` to guess who.", quote, length)
	p.b.Send(r.Conn, bot.Message, r.Msg.Channel, msg)

	return true
}

func (p *QuoteGame) guess(r bot.Request) bool {
	log.Debug().Msg("quote game message check")
	if p.currentGame == nil {
		return false
	}
	if r.Values["name"] == p.currentName {
		msg := fmt.Sprintf("%s won the quote game!", r.Msg.User.Name)
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel, msg)
		p.currentName = ""
		p.currentGame.Stop()
		p.currentGame = nil
		return true
	}

	p.b.Send(r.Conn, bot.Message, r.Msg.Channel,
		fmt.Sprintf("Sorry %s, that's not correct.", r.Msg.User.Name))

	return true
}
