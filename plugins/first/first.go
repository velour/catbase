// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package first

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
)

// This is a first plugin to serve as an example and quick copy/paste for new plugins.

type FirstPlugin struct {
	bot      bot.Bot
	db       *sqlx.DB
	handlers bot.HandlerTable
	enabled  bool
}

type FirstEntry struct {
	id      int64
	day     time.Time
	time    time.Time
	channel string
	body    string
	nick    string
	saved   bool
}

// Insert or update the first entry
func (fe *FirstEntry) save(db *sqlx.DB) error {
	if _, err := db.Exec(`insert into first (day, time, channel, body, nick)
		values (?, ?, ?, ?, ?)`,
		fe.day.Unix(),
		fe.time.Unix(),
		fe.channel,
		fe.body,
		fe.nick,
	); err != nil {
		return err
	}
	return nil
}

func (fe *FirstEntry) delete(db *sqlx.DB) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`delete from first where id=?`, fe.id)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

// NewFirstPlugin creates a new FirstPlugin with the Plugin interface
func New(b bot.Bot) *FirstPlugin {
	_, err := b.DB().Exec(`create table if not exists first (
			id integer primary key,
			day integer,
			time integer,
			channel string,
			body string,
			nick string
		);`)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Could not create first table")
	}

	log.Info().Msgf("First plugin initialized with day: %s",
		midnight(time.Now()))

	fp := &FirstPlugin{
		bot:     b,
		db:      b.DB(),
		enabled: true,
	}
	fp.register()
	b.Register(fp, bot.Help, fp.help)
	return fp
}

func getLastFirst(db *sqlx.DB, channel string) (*FirstEntry, error) {
	// Get last first entry
	var id sql.NullInt64
	var day sql.NullInt64
	var timeEntered sql.NullInt64
	var body sql.NullString
	var nick sql.NullString

	err := db.QueryRow(`select
		id, max(day), time, body, nick from first
		where channel = ?
		limit 1;
	`, channel).Scan(
		&id,
		&day,
		&timeEntered,
		&body,
		&nick,
	)
	switch {
	case err == sql.ErrNoRows || !id.Valid:
		log.Info().Msg("No previous first entries")
		return nil, nil
	case err != nil:
		log.Warn().Err(err).Msg("Error on first query row")
		return nil, err
	}
	log.Debug().Msgf("id: %v day %v time %v body %v nick %v",
		id, day, timeEntered, body, nick)
	return &FirstEntry{
		id:      id.Int64,
		day:     time.Unix(day.Int64, 0),
		time:    time.Unix(timeEntered.Int64, 0),
		channel: channel,
		body:    body.String,
		nick:    nick.String,
		saved:   true,
	}, nil
}

func midnight(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func isNotToday(f *FirstEntry) bool {
	if f == nil {
		return true
	}
	t := f.time
	t0 := midnight(t)
	return t0.Before(midnight(time.Now()))
}

func (p *FirstPlugin) register() {
	p.handlers = []bot.HandlerSpec{
		{Kind: bot.Message, IsCmd: false,
			Regex: regexp.MustCompile(`(?i)^who'?s on first the most.?$`),
			Handler: func(r bot.Request) bool {
				first, err := getLastFirst(p.db, r.Msg.Channel)
				if first != nil && err == nil {
					p.leaderboard(r.Conn, r.Msg.Channel)
					return true
				}
				return false
			}},
		{Kind: bot.Message, IsCmd: false,
			Regex: regexp.MustCompile(`(?i)^who'?s on first.?$`),
			Handler: func(r bot.Request) bool {
				first, err := getLastFirst(p.db, r.Msg.Channel)
				if first != nil && err == nil {
					p.announceFirst(r.Conn, first)
					return true
				}
				return false
			}},
		{Kind: bot.Message, IsCmd: true,
			Regex: regexp.MustCompile(`(?i)^clear first$`),
			Handler: func(r bot.Request) bool {
				if !p.bot.CheckAdmin(r.Msg.User.Name) {
					p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, "You are not authorized to do that.")
					return true
				}
				fe, err := getLastFirst(p.db, r.Msg.Channel)
				if err != nil {
					p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, "Could not find a first entry.")
					return true
				}
				p.enabled = false
				err = fe.delete(p.db)
				if err != nil {
					p.bot.Send(r.Conn, bot.Message, r.Msg.Channel,
						fmt.Sprintf("Could not delete first entry: %s", err))
					p.enabled = true
					return true
				}
				d := p.bot.Config().GetInt("first.maxregen", 300)
				log.Debug().Msgf("Setting first timer for %d seconds", d)
				timer := time.NewTimer(time.Duration(d) * time.Second)
				go func() {
					<-timer.C
					p.enabled = true
					log.Debug().Msgf("Re-enabled first")
				}()
				p.bot.Send(r.Conn, bot.Message, r.Msg.Channel,
					fmt.Sprintf("Deleted first entry: '%s' and set a random timer for when first will happen next.", fe.body))
				return true
			}},
		{Kind: bot.Message, IsCmd: false,
			Regex: regexp.MustCompile(`.*`),
			Handler: func(r bot.Request) bool {
				if r.Msg.IsIM || !p.enabled {
					return false
				}

				first, err := getLastFirst(p.db, r.Msg.Channel)
				if err != nil {
					log.Error().
						Err(err).
						Msg("Error getting last first")
				}

				log.Debug().Bool("first == nil", first == nil).Msg("Is first nil?")
				log.Debug().Bool("first == nil || isNotToday()", isNotToday(first)).Msg("Is it today?")
				log.Debug().Bool("p.allowed", p.allowed(r.Msg)).Msg("Allowed?")

				if (first == nil || isNotToday(first)) && p.allowed(r.Msg) {
					log.Debug().
						Str("body", r.Msg.Body).
						Interface("t0", first).
						Time("t1", time.Now()).
						Msg("Recording first")
					p.recordFirst(r.Conn, r.Msg)
					return false
				}
				return false
			}},
	}
	p.bot.RegisterTable(p, p.handlers)
}

func (p *FirstPlugin) allowed(message msg.Message) bool {
	if message.Body == "" {
		return false
	}
	for _, m := range p.bot.Config().GetArray("Bad.Msgs", []string{}) {
		match, err := regexp.MatchString(m, strings.ToLower(message.Body))
		if err != nil {
			log.Error().Err(err).Msg("Bad regexp")
		}
		if match {
			log.Info().
				Str("user", message.User.Name).
				Str("body", message.Body).
				Msg("Disallowing first")
			return false
		}
	}
	for _, host := range p.bot.Config().GetArray("Bad.Hosts", []string{}) {
		if host == message.Host {
			log.Info().
				Str("user", message.User.Name).
				Str("body", message.Body).
				Msg("Disallowing first")
			return false
		}
	}
	for _, nick := range p.bot.Config().GetArray("Bad.Nicks", []string{}) {
		if nick == message.User.Name {
			log.Info().
				Str("user", message.User.Name).
				Str("body", message.Body).
				Msg("Disallowing first")
			return false
		}
	}
	return true
}

func (p *FirstPlugin) recordFirst(c bot.Connector, message msg.Message) {
	log.Info().
		Str("channel", message.Channel).
		Str("user", message.User.Name).
		Str("body", message.Body).
		Msg("Recording first")
	first := &FirstEntry{
		day:     midnight(time.Now()),
		time:    message.Time,
		channel: message.Channel,
		body:    message.Body,
		nick:    message.User.Name,
	}
	log.Info().Msgf("recordFirst: %+v", first.day)
	err := first.save(p.db)
	if err != nil {
		log.Error().Err(err).Msg("Error saving first entry")
		return
	}
	p.announceFirst(c, first)
}

func (p *FirstPlugin) leaderboard(c bot.Connector, ch string) error {
	q := `select max(channel) channel, max(nick) nick, count(id) count
		from first
		group by channel, nick
		having channel = ?
		order by count desc
		limit 3`
	res := []struct {
		Channel string
		Nick    string
		Count   int
	}{}
	err := p.db.Select(&res, q, ch)
	if err != nil {
		return err
	}
	talismans := []string{":gold-trophy:", ":silver-trophy:", ":bronze-trophy:"}
	msg := "First leaderboard:\n"
	for i, e := range res {
		msg += fmt.Sprintf("%s %d %s\n", talismans[i], e.Count, e.Nick)
	}
	p.bot.Send(c, bot.Message, ch, msg)
	return nil
}

func (p *FirstPlugin) announceFirst(c bot.Connector, first *FirstEntry) {
	ch := first.channel
	p.bot.Send(c, bot.Message, ch, fmt.Sprintf("%s had first at %s with the message: \"%s\"",
		first.nick, first.time.Format("15:04"), first.body))
}

// Help responds to help requests. Every plugin must implement a help function.
func (p *FirstPlugin) help(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	p.bot.Send(c, bot.Message, message.Channel, "You can ask 'who's on first?' to find out.")
	return true
}
