package quotegame

import (
	"math/rand"
	"regexp"
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
			Kind: bot.Message, IsCmd: true,
			Regex:    regexp.MustCompile(`(?i)^quote game$`),
			HelpText: "Start a quote game",
			Handler:  p.startGame,
		},
		{
			Kind: bot.Message, IsCmd: false,
			Regex:    regexp.MustCompile(`.*`),
			Handler:  p.message,
		},
	}
	p.b.RegisterTable(p, p.handlers)
}

func (p *QuoteGame) getAllQuotes() ([]string, error) {
	threshold := p.c.GetInt("quotegame.threshold", 10)
	q := `
	select tidbit from fact where fact in (
		select fact, verb, tidbit from fact where fact like '%quotes' group by fact having count(fact) > ?
	)
	`
	quotes := []string{}
	err := p.db.Select(&quotes, q, threshold)
	if err != nil {
		return nil, err
	}
	return quotes, nil
}

func (p *QuoteGame) getRandomquote() (string, error) {
	quotes, err := p.getAllQuotes()
	if err != nil {
		return "", err
	}
	return quotes[rand.Intn(len(quotes))], nil
}

func (p *QuoteGame) startGame(r bot.Request) bool {
	log.Debug().Msg("startGame called")
	if p.currentGame != nil {
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "There is already a quote game running.")
		return true
	}

	length := time.Duration(p.c.GetInt("quotegame.length", 120))
	p.currentGame = time.AfterFunc(length * time.Second, func() {
		p.currentGame = nil
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "Game ended.")
	})

	p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "Game started.")

	return true
}

func (p *QuoteGame) message(r bot.Request) bool {
	if p.currentGame == nil {
		return false
	}
	return false
}
