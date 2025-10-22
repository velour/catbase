// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package reminder

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"code.chrissexton.org/cws/catbaseduration"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"

	"github.com/olebedev/when"
	"github.com/olebedev/when/rules/common"
	"github.com/olebedev/when/rules/en"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/config"
	"github.com/velour/catbase/plugins/sms"
)

const (
	TIMESTAMP = "2006-01-02 15:04:05"
)

type ReminderPlugin struct {
	bot          bot.Bot
	db           *sqlx.DB
	mutex        *sync.Mutex
	timer        *time.Timer
	config       *config.Config
	when         *when.Parser
	lastReminder map[string]*Reminder
}

type Reminder struct {
	id      int64
	from    string
	who     string
	what    string
	when    time.Time
	channel string
}

func New(b bot.Bot) *ReminderPlugin {
	if _, err := b.DB().Exec(`create table if not exists reminders (
			id integer primary key,
			fromWho string,
			toWho string,
			what string,
			remindWhen string,
			channel string
		);`); err != nil {
		log.Fatal().Err(err)
	}

	dur, _ := time.ParseDuration("1h")
	timer := time.NewTimer(dur)
	timer.Stop()

	w := when.New(nil)
	w.Add(en.All...)
	w.Add(common.All...)

	plugin := &ReminderPlugin{
		bot:          b,
		db:           b.DB(),
		mutex:        &sync.Mutex{},
		timer:        timer,
		config:       b.Config(),
		when:         w,
		lastReminder: map[string]*Reminder{},
	}

	plugin.queueUpNextReminder()

	go plugin.reminderer(b.DefaultConnector())

	b.RegisterRegexCmd(plugin, bot.Message, regexp.MustCompile(`(?i)^snooze (?P<duration>.+)$`), plugin.snooze)
	b.Register(plugin, bot.Message, plugin.message)
	b.Register(plugin, bot.Help, plugin.help)

	return plugin
}

func (p *ReminderPlugin) snooze(r bot.Request) bool {
	lastReminder := p.lastReminder[r.Msg.Channel]
	if lastReminder == nil {
		p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, "My memory is too small to contain a snoozed reminder.")
		return true
	}
	durationTxt := replaceDuration(p.when, r.Values["duration"])
	dur, err := catbaseduration.ParseDuration(durationTxt)
	if err != nil {
		p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, "Whoa, cowboy. I can't parse that time duration.")
		return true
	}
	lastReminder.when = time.Now().UTC().Add(dur)
	p.addReminder(lastReminder)
	delete(p.lastReminder, r.Msg.Channel)
	p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("Okay, I'll let you know in %s", dur))
	p.queueUpNextReminder()
	return true
}

func replaceDuration(when *when.Parser, txt string) string {
	t, err := when.Parse(txt, time.Now())
	if t != nil && err == nil {
		return txt[0:t.Index] + t.Time.Sub(time.Now()).String() + txt[t.Index+len(t.Text):]
	}
	return txt
}

func (p *ReminderPlugin) message(c bot.Connector, kind bot.Kind, message msg.Message, args ...any) bool {
	channel := message.Channel
	from := message.User.Name

	parts := strings.Fields(message.Body)

	if len(parts) >= 5 {
		if strings.ToLower(parts[0]) == "remind" {
			who := parts[1]
			if who == "me" {
				who = from
			}

			dur, err := catbaseduration.ParseDuration(parts[3])
			if err != nil {
				p.bot.Send(c, bot.Message, channel, "Easy cowboy, not sure I can parse that duration. Try something like '1.5h' or '2h45m'. You can use yMd for year/month/day and Go durations otherwise.")
				return true
			}

			operator := strings.ToLower(parts[2])
			doConfirm := true

			if operator == "in" || operator == "at" || operator == "on" {
				//one off reminder
				//remind who in dur blah
				when := time.Now().UTC().Add(dur)
				what := strings.Join(parts[4:], " ")

				p.addReminder(&Reminder{
					id:      -1,
					from:    from,
					who:     who,
					what:    what,
					when:    when,
					channel: channel,
				})

			} else if operator == "every" && strings.ToLower(parts[4]) == "for" {
				//batch add, especially for reminding msherms to buy a kit
				//remind who every dur for dur2 blah
				dur2, err := catbaseduration.ParseDuration(parts[5])
				if err != nil {
					log.Error().Err(err)
					p.bot.Send(c, bot.Message, channel, "Easy cowboy, not sure I can parse that duration. Try something like '1.5h' or '2h45m'.")
					return true
				}

				when := time.Now().UTC().Add(dur)
				endTime := time.Now().UTC().Add(dur2)
				what := strings.Join(parts[6:], " ")

				max := p.config.GetInt("Reminder.MaxBatchAdd", 10)
				for i := 0; when.Before(endTime); i++ {
					if i >= max {
						p.bot.Send(c, bot.Message, channel, "Easy cowboy, that's a lot of reminders. I'll add some of them.")
						doConfirm = false
						break
					}

					p.addReminder(&Reminder{
						id:      int64(-1),
						from:    from,
						who:     who,
						what:    what,
						when:    when,
						channel: channel,
					})

					when = when.Add(dur)
				}
			} else {
				p.bot.Send(c, bot.Message, channel, "Easy cowboy, not sure I comprehend what you're asking.")
				return true
			}

			if doConfirm && from == who {
				p.bot.Send(c, bot.Message, channel, fmt.Sprintf("Okay. I'll remind you."))
			} else if doConfirm {
				p.bot.Send(c, bot.Message, channel, fmt.Sprintf("Sure %s, I'll remind %s.", from, who))
			}

			p.queueUpNextReminder()

			return true
		}
	} else if len(parts) >= 2 && strings.ToLower(parts[0]) == "list" && strings.ToLower(parts[1]) == "reminders" {
		var response string
		var err error
		if len(parts) == 2 {
			response, err = p.getAllRemindersFormatted(channel)
		} else if len(parts) == 4 {
			if strings.ToLower(parts[2]) == "to" {
				response, err = p.getAllRemindersToMeFormatted(channel, strings.ToLower(parts[3]))
			} else if strings.ToLower(parts[2]) == "from" {
				response, err = p.getAllRemindersFromMeFormatted(channel, strings.ToLower(parts[3]))
			}
		}
		if err != nil {
			p.bot.Send(c, bot.Message, channel, "listing failed.")
		} else {
			p.bot.Send(c, bot.Message, channel, response)
		}
		return true
	} else if len(parts) == 3 && strings.ToLower(parts[0]) == "cancel" && strings.ToLower(parts[1]) == "reminder" {
		id, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			p.bot.Send(c, bot.Message, channel, fmt.Sprintf("couldn't parse id: %s", parts[2]))

		} else {
			err := p.deleteReminder(id)
			if err == nil {
				p.bot.Send(c, bot.Message, channel, fmt.Sprintf("successfully canceled reminder: %s", parts[2]))
			} else {
				p.bot.Send(c, bot.Message, channel, fmt.Sprintf("failed to find and cancel reminder: %s", parts[2]))
			}
		}
		return true
	}

	return false
}

func (p *ReminderPlugin) help(c bot.Connector, kind bot.Kind, message msg.Message, args ...any) bool {
	p.bot.Send(c, bot.Message, message.Channel, "Pester someone with a reminder. Try \"remind <user> in <duration> message\".\n\nUnsure about duration syntax? Check https://golang.org/pkg/time/#ParseDuration\n\nYou can also use y/M/d for year/month/day. Months count as 30 days.")
	return true
}

func (p *ReminderPlugin) getNextReminder() *Reminder {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	rows, err := p.db.Query("select id, fromWho, toWho, what, remindWhen, channel from reminders order by remindWhen asc limit 1;")
	if err != nil {
		log.Error().Err(err)
		return nil
	}
	defer rows.Close()

	once := false
	var reminder *Reminder
	for rows.Next() {
		if once {
			log.Debug().Msg("somehow got multiple rows")
		}
		reminder = &Reminder{}

		var when string
		err := rows.Scan(&reminder.id, &reminder.from, &reminder.who, &reminder.what, &when, &reminder.channel)
		if err != nil {
			log.Error().Err(err)
			return nil
		}
		reminder.when, err = time.Parse(TIMESTAMP, when)
		if err != nil {
			log.Error().Err(err)
			return nil
		}

		once = true
	}

	return reminder
}

func (p *ReminderPlugin) addReminder(reminder *Reminder) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	_, err := p.db.Exec(`insert into reminders (fromWho, toWho, what, remindWhen, channel) values (?, ?, ?, ?, ?);`,
		reminder.from, reminder.who, reminder.what, reminder.when.Format(TIMESTAMP), reminder.channel)

	if err != nil {
		log.Error().Err(err)
	}
	return err
}

func (p *ReminderPlugin) deleteReminder(id int64) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	res, err := p.db.Exec(`delete from reminders where id = ?;`, id)
	if err != nil {
		log.Error().Err(err)
	} else {
		if affected, err := res.RowsAffected(); err != nil {
			return err
		} else if affected != 1 {
			return errors.New("didn't delete any rows")
		}
	}
	return err
}

func (p *ReminderPlugin) getRemindersFormatted(filter string) (string, error) {
	max := p.config.GetInt("Reminder.MaxList", 25)
	queryString := fmt.Sprintf("select id, fromWho, toWho, what, remindWhen from reminders %s order by remindWhen asc limit %d;", filter, max)
	countString := fmt.Sprintf("select COUNT(*) from reminders %s;", filter)

	p.mutex.Lock()
	defer p.mutex.Unlock()

	var total int
	err := p.db.Get(&total, countString)
	if err != nil {
		log.Error().Err(err)
		return "", nil
	}

	if total == 0 {
		return "no pending reminders", nil
	}

	rows, err := p.db.Query(queryString)
	if err != nil {
		log.Error().Err(err)
		return "", nil
	}
	defer rows.Close()
	reminders := ""
	counter := 1
	reminder := &Reminder{}
	for rows.Next() {
		var when string
		err := rows.Scan(&reminder.id, &reminder.from, &reminder.who, &reminder.what, &when)
		if err != nil {
			return "", err
		}
		reminders += fmt.Sprintf("%d) %s -> %s :: %s @ %s (%d)\n", counter, reminder.from, reminder.who, reminder.what, when, reminder.id)
		counter++
	}

	remaining := total - max
	if remaining > 0 {
		reminders += fmt.Sprintf("...%d more...\n", remaining)
	}

	return reminders, nil
}

func (p *ReminderPlugin) getAllRemindersFormatted(channel string) (string, error) {
	return p.getRemindersFormatted("")
}

func (p *ReminderPlugin) getAllRemindersFromMeFormatted(channel, me string) (string, error) {
	return p.getRemindersFormatted(fmt.Sprintf("where fromWho = '%s'", me))
}

func (p *ReminderPlugin) getAllRemindersToMeFormatted(channel, me string) (string, error) {
	return p.getRemindersFormatted(fmt.Sprintf("where toWho = '%s'", me))
}

func (p *ReminderPlugin) queueUpNextReminder() {
	nextReminder := p.getNextReminder()

	if nextReminder != nil {
		p.timer.Reset(nextReminder.when.Sub(time.Now().UTC()))
	}
}

func (p *ReminderPlugin) reminderer(c bot.Connector) {

	for {
		<-p.timer.C

		reminder := p.getNextReminder()

		if reminder != nil && time.Now().UTC().After(reminder.when) {
			p.lastReminder[reminder.channel] = reminder
			var message string
			if reminder.from == reminder.who {
				reminder.from = "you"
				message = fmt.Sprintf("Hey %s, you wanted to be reminded: %s", reminder.who, reminder.what)
			} else {
				message = fmt.Sprintf("Hey %s, %s wanted you to be reminded: %s", reminder.who, reminder.from, reminder.what)
			}

			p.bot.Send(c, bot.Message, reminder.channel, message)
			smsPlugin := sms.New(p.bot)
			if err := smsPlugin.Send(reminder.who, message); err != nil {
				//log.Error().Err(err).Msgf("could not send reminder")
			}

			if err := p.deleteReminder(reminder.id); err != nil {
				log.Error().
					Int64("id", reminder.id).
					Err(err).
					Msg("this will cause problems, we need to stop now.")
			}
		}

		p.queueUpNextReminder()
	}
}
