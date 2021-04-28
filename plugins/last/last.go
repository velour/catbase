package last

import (
	"fmt"
	"regexp"
	"time"

	"github.com/velour/catbase/config"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/plugins/first"
)

type LastPlugin struct {
	b  bot.Bot
	db *sqlx.DB
	c  *config.Config

	handlers bot.HandlerTable
	channels map[string]bool
}

func New(b bot.Bot) *LastPlugin {
	p := &LastPlugin{
		b:        b,
		db:       b.DB(),
		c:        b.Config(),
		channels: map[string]bool{},
	}
	if err := p.migrate(); err != nil {
		panic(err)
	}
	p.register()
	return p
}

func (p *LastPlugin) migrate() error {
	tx, err := p.db.Beginx()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`create table if not exists last (
    	day integer,
        channel string not null,
        ts int not null,
    	who string not null,
    	message string not null,
    	constraint last_key primary key (day, channel) on conflict replace
	)`)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (p *LastPlugin) register() {
	p.handlers = bot.HandlerTable{
		{
			Kind: bot.Message, IsCmd: true,
			Regex:    regexp.MustCompile(`(?i)^who killed the channel\??$`),
			HelpText: "Find out who had last yesterday",
			Handler:  p.whoKilled,
		},
		{
			Kind: bot.Message, IsCmd: true,
			Regex:    regexp.MustCompile(`(?i)^who killed #?(?P<channel>\S+)\??$`),
			HelpText: "Find out who had last yesterday in a channel",
			Handler:  p.whoKilledChannel,
		},
		{
			Kind: bot.Message, IsCmd: false,
			Regex:    regexp.MustCompile(`.*`),
			HelpText: "Last does secret stuff you don't need to know about.",
			Handler:  p.recordLast,
		},
	}
	p.b.RegisterTable(p, p.handlers)
}

func nextNoon(t time.Time) time.Duration {
	day := first.Midnight(t)
	nextNoon := day.Add(12 * time.Hour)
	log.Debug().
		Time("t", t).
		Time("nextNoon", nextNoon).
		Bool("before(t)", nextNoon.Before(t)).
		Msgf("nextNoon")
	if nextNoon.Before(t) {
		nextNoon = nextNoon.Add(24 * time.Hour)
	}
	log.Debug().Msgf("nextNoon.Sub(t): %v", nextNoon.Sub(t))
	return nextNoon.Sub(t)
}

func (p *LastPlugin) recordLast(r bot.Request) bool {
	ch := r.Msg.Channel
	who := r.Msg.User.Name
	day := first.Midnight(time.Now())

	if _, ok := p.channels[ch]; !ok {
		if !p.b.OnBlacklist(ch, bot.PluginName(p)) {
			p.channels[ch] = true
			log.Debug().Msgf("Next Noon: %v", nextNoon(time.Now().UTC()))
			time.AfterFunc(nextNoon(time.Now().Local()), p.reportLast(ch))
		}
	}

	if r.Msg.Body == "" {
		return false
	}

	invalidUsers := p.c.GetArray("last.invalidUsers", []string{"unknown"})
	for _, u := range invalidUsers {
		if who == u {
			return false
		}
	}

	_, err := p.db.Exec(
		`insert into last values (?, ?, ?, ?, ?)`,
		day.Unix(), ch, time.Now().Unix(), who, r.Msg.Body)
	if err != nil {
		log.Error().Err(err).Msgf("Could not record last.")
	}
	return false
}

type last struct {
	Day     int64
	TS      int64
	Channel string
	Who     string
	Message string
}

func (p *LastPlugin) yesterdaysLast(ch string) (last, error) {
	l := last{}
	midnight := first.Midnight(time.Now())
	q := `select * from last where channel = ? and day < ?  and day >= ? order by day limit 1`
	log.Debug().Str("q", q).Msgf("yesterdaysLast: %d to %d", midnight.Unix(), midnight.Add(-24*time.Hour).Unix())
	err := p.db.Get(&l, q, ch, midnight.Unix(), midnight.Add(-24*time.Hour).Unix())
	if err != nil {
		return l, err
	}
	return l, nil
}

func (p *LastPlugin) reportLast(ch string) func() {
	return func() {
		p.sayLast(p.b.DefaultConnector(), ch, ch)
		time.AfterFunc(24*time.Hour, p.reportLast(ch))
	}
}

func (p *LastPlugin) whoKilled(r bot.Request) bool {
	p.sayLast(r.Conn, r.Msg.Channel, r.Msg.Channel)
	return true
}

func (p *LastPlugin) whoKilledChannel(r bot.Request) bool {
	ch := r.Values["channel"]
	p.sayLast(r.Conn, r.Conn.GetChannelID(ch), r.Msg.Channel)
	return true
}

func (p *LastPlugin) sayLast(c bot.Connector, chFrom, chTo string) {
	l, err := p.yesterdaysLast(chFrom)
	if err != nil {
		log.Error().Err(err).Msgf("Couldn't find last")
		p.b.Send(c, bot.Message, chTo, "I couldn't find a last.")
		return
	}
	if l.Day == 0 {
		log.Error().Interface("l", l).Msgf("Couldn't find last")
		p.b.Send(c, bot.Message, chTo, "I couldn't find a last.")
	}
	msg := fmt.Sprintf(`%s killed the channel last night by saying "%s"`, l.Who, l.Message)
	p.b.Send(c, bot.Message, chTo, msg)
}
