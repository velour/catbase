package newsbid

import (
	"github.com/jmoiron/sqlx"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
)

type NewsBid struct {
	bot bot.Bot
	db  *sqlx.DB
}

func New(b bot.Bot) *NewsBid {
	p := &NewsBid{
		bot: b,
		db:  b.DB(),
	}
	p.bot.Register(p, bot.Message, p.message)
	return p
}

func (p *NewsBid) message(conn bot.Connector, k bot.Kind, message msg.Message, args ...interface{}) bool {
	return false
}
