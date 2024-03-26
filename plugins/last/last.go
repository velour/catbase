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
        time int not null,
    	nick string not null,
    	body string not null,
    	message_id string not null,
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
			Kind: bot.Any, IsCmd: false,
			Regex:    regexp.MustCompile(`.*`),
			HelpText: "Last does secret stuff you don't need to know about.",
			Handler:  p.recordLast,
		},
	}
	p.b.RegisterTable(p, p.handlers)
}

func (p *LastPlugin) enabled_channel(r bot.Request) bool {
	chs := p.c.GetArray("last.channels", []string{})
	for _, ch := range chs {
		if r.Msg.Channel == ch {
			return true
		}
	}
	return false
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
	if !p.enabled_channel(r) {
		return false
	}
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
		`insert into last (day, channel, time, body, nick, message_id) values (?, ?, ?, ?, ?, ?)`,
		day.Unix(), ch, time.Now().Unix(), r.Msg.Body, who, r.Msg.ID)
	if err != nil {
		log.Error().Err(err).Msgf("Could not record last.")
	}
	return false
}

type last struct {
	ID        int64  `db:"id"`
	Day       int64  `db:"day"`
	Time      int64  `db:"time"`
	Channel   string `db:"channel"`
	Nick      string `db:"nick"`
	Body      string `db:"body"`
	MessageID string `db:"message_id"`
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
		p.sayLast(p.b.DefaultConnector(), ch, ch, false)
		time.AfterFunc(24*time.Hour, p.reportLast(ch))
	}
}

func (p *LastPlugin) whoKilled(r bot.Request) bool {
	p.sayLast(r.Conn, r.Msg.Channel, r.Msg.Channel, true)
	return true
}

func (p *LastPlugin) whoKilledChannel(r bot.Request) bool {
	ch := r.Values["channel"]
	p.sayLast(r.Conn, r.Conn.GetChannelID(ch), r.Msg.Channel, true)
	return true
}

func (p *LastPlugin) sayLast(c bot.Connector, chFrom, chTo string, force bool) {
	l, err := p.yesterdaysLast(chFrom)
	if err != nil || l.Day == 0 {
		log.Error().Err(err).Interface("last", l).Msgf("Couldn't find last")
		if force {
			p.b.Send(c, bot.Message, chTo, "I couldn't find a last.")
		}
		return
	}
	timeOfDay := "last night"
	hour := time.Unix(l.Time, 0).Hour()
	if hour < 18 {
		timeOfDay = "in the afternoon"
	}
	if hour < 12 {
		timeOfDay = "in the morning"
	}
	log.Debug().
		Str("timeOfDay", timeOfDay).
		Int("hour", hour).
		Int64("l.Time", l.Time).
		Msgf("killed")
	msg := fmt.Sprintf(`%s killed the channel %s`, l.Nick, timeOfDay)
	guildID := p.c.Get("discord.guildid", "")
	p.b.Send(c, bot.Message, chTo, msg, bot.MessageReference{
		MessageID: l.MessageID,
		ChannelID: l.Channel,
		GuildID:   guildID,
	})
}
