package quotegame

import (
	"math/rand"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/config"
)

type QuoteGame struct {
	b  bot.Bot
	c  *config.Config
	db *sqlx.DB

	currentGame *time.Timer
}

func New(b bot.Bot) *QuoteGame {
	return &QuoteGame{
		b:           b,
		c:           b.Config(),
		db:          b.DB(),
		currentGame: nil,
	}
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