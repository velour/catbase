// Countdown is a plugin to do tasks on various milestones.
package countdown

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/config"
)

var nextYear = time.Date(time.Now().Year()+1, 0, 0, 0, 0, 0, 0, time.Local)

type CountdownPlugin struct {
	b  bot.Bot
	db *sqlx.DB
	c  *config.Config
}

func New(b bot.Bot) *CountdownPlugin {
	p := &CountdownPlugin{
		b:  b,
		db: b.DB(),
		c:  b.Config(),
	}

	p.scheduleNYE(p.newYearRollover)

	return p
}

func (p *CountdownPlugin) scheduleNYE(f func()) {
	time.AfterFunc(nextYear.Sub(time.Now()), f)
	for i := 10; i > 0; i-- {
		mkFunc := func(i int) func() {
			return func() {
				p.nyeCountdown(i)
			}
		}
		time.AfterFunc(nextYear.Add(time.Duration(10-i)*time.Second).Sub(time.Now()), mkFunc(i))
	}
}

func (p *CountdownPlugin) newYearRollover() {
	log.Debug().Msgf("Happy new year!")

	tx, err := p.db.Beginx()
	if err != nil {
		logError(err, "error getting transaction")
		return
	}
	year := time.Now().Year() - 1
	q := fmt.Sprintf(`insert into counter_%d select * from counter`, year)
	_, err = tx.Exec(q)
	if err != nil {
		logError(err, "error running insert into")
		logError(tx.Rollback(), "error with insert rollback")
		return
	}
	q = `delete from counter`
	_, err = tx.Exec(q)
	if err != nil {
		logError(err, "error running delete")
		logError(tx.Rollback(), "error with delete rollback")
		return
	}
	err = tx.Commit()
	logError(err, "error committing transaction")
}

func logError(err error, msg string) {
	if err == nil {
		return
	}
	log.Error().Err(err).Msgf(msg)
}

func (p *CountdownPlugin) nyeCountdown(i int) {
	chs := p.c.GetArray("channels", []string{})
	for _, ch := range chs {
		msg := fmt.Sprintf("%d!", i)
		p.b.Send(p.b.DefaultConnector(), bot.Message, ch, msg, msg)
	}
}
