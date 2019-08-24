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
	Bot bot.Bot
	db  *sqlx.DB
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
		Bot: b,
		db:  b.DB(),
	}
	b.Register(fp, bot.Message, fp.message)
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

// Message responds to the bot hook on recieving messages.
// This function returns true if the plugin responds in a meaningful way to the users message.
// Otherwise, the function returns false and the bot continues execution of other plugins.
func (p *FirstPlugin) message(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	log.Debug().
		Interface("msg", message).
		Msg("First is looking at a message")

	if message.IsIM {
		log.Debug().Msg("Skipping IM")
		return false
	}

	first, err := getLastFirst(p.db, message.Channel)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Error getting last first")
	}

	log.Debug().Bool("first == nil", first == nil).Msg("Is first nil?")
	log.Debug().Bool("first == nil || isNotToday()", isNotToday(first)).Msg("Is it today?")
	log.Debug().Bool("p.allowed", p.allowed(message)).Msg("Allowed?")

	if (first == nil || isNotToday(first)) && p.allowed(message) {
		log.Debug().
			Str("body", message.Body).
			Interface("t0", first).
			Time("t1", time.Now()).
			Msg("Recording first")
		p.recordFirst(c, message)
		return false
	}

	r := strings.NewReplacer("’", "", "'", "", "\"", "", ",", "", ".", "", ":", "",
		"?", "", "!", "")
	m := strings.ToLower(message.Body)
	if r.Replace(m) == "whos on first" && first != nil {
		p.announceFirst(c, first)
		return true
	}

	return false
}

func (p *FirstPlugin) allowed(message msg.Message) bool {
	for _, m := range p.Bot.Config().GetArray("Bad.Msgs", []string{}) {
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
	for _, host := range p.Bot.Config().GetArray("Bad.Hosts", []string{}) {
		if host == message.Host {
			log.Info().
				Str("user", message.User.Name).
				Str("body", message.Body).
				Msg("Disallowing first")
			return false
		}
	}
	for _, nick := range p.Bot.Config().GetArray("Bad.Nicks", []string{}) {
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

func (p *FirstPlugin) announceFirst(c bot.Connector, first *FirstEntry) {
	ch := first.channel
	p.Bot.Send(c, bot.Message, ch, fmt.Sprintf("%s had first at %s with the message: \"%s\"",
		first.nick, first.time.Format("15:04"), first.body))
}

// Help responds to help requests. Every plugin must implement a help function.
func (p *FirstPlugin) help(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	p.Bot.Send(c, bot.Message, message.Channel, "You can ask 'who's on first?' to find out.")
	return true
}
