// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package reminder

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/config"
)

const (
	TIMESTAMP = "2006-01-02 15:04:05"
)

type ReminderPlugin struct {
	Bot    bot.Bot
	db     *sqlx.DB
	mutex  *sync.Mutex
	timer  *time.Timer
	config *config.Config
}

type Reminder struct {
	id      int64
	from    string
	who     string
	what    string
	when    time.Time
	channel string
}

func New(bot bot.Bot) *ReminderPlugin {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	if _, err := bot.DB().Exec(`create table if not exists reminders (
			id integer primary key,
			fromWho string,
			toWho string,
			what string,
			remindWhen string,
			channel string
		);`); err != nil {
		log.Fatal(err)
	}

	dur, _ := time.ParseDuration("1h")
	timer := time.NewTimer(dur)
	timer.Stop()

	plugin := &ReminderPlugin{
		Bot:    bot,
		db:     bot.DB(),
		mutex:  &sync.Mutex{},
		timer:  timer,
		config: bot.Config(),
	}

	plugin.queueUpNextReminder()

	go reminderer(plugin)

	return plugin
}

func (p *ReminderPlugin) Message(message msg.Message) bool {
	channel := message.Channel
	from := message.User.Name

	parts := strings.Fields(message.Body)

	if len(parts) >= 5 {
		if strings.ToLower(parts[0]) == "remind" {
			who := parts[1]
			if who == "me" {
				who = from
			}

			dur, err := time.ParseDuration(parts[3])
			if err != nil {
				p.Bot.SendMessage(channel, "Easy cowboy, not sure I can parse that duration.")
				return true
			}

			operator := strings.ToLower(parts[2])

			doConfirm := true

			if operator == "in" {
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
				dur2, err := time.ParseDuration(parts[5])
				if err != nil {
					p.Bot.SendMessage(channel, "Easy cowboy, not sure I can parse that duration.")
					return true
				}

				when := time.Now().UTC().Add(dur)
				endTime := time.Now().UTC().Add(dur2)
				what := strings.Join(parts[6:], " ")

				for i := 0; when.Before(endTime); i++ {
					max := p.config.GetInt("Reminder.MaxBatchAdd")
					if max == 0 {
						max = 10
						p.config.Set("reminder.maxbatchadd", strconv.Itoa(max))
					}
					if i >= max {
						p.Bot.SendMessage(channel, "Easy cowboy, that's a lot of reminders. I'll add some of them.")
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
				p.Bot.SendMessage(channel, "Easy cowboy, not sure I comprehend what you're asking.")
				return true
			}

			if doConfirm && from == who {
				p.Bot.SendMessage(channel, fmt.Sprintf("Okay. I'll remind you."))
			} else if doConfirm {
				p.Bot.SendMessage(channel, fmt.Sprintf("Sure %s, I'll remind %s.", from, who))
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
			p.Bot.SendMessage(channel, "listing failed.")
		} else {
			p.Bot.SendMessage(channel, response)
		}
		return true
	} else if len(parts) == 3 && strings.ToLower(parts[0]) == "cancel" && strings.ToLower(parts[1]) == "reminder" {
		id, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			p.Bot.SendMessage(channel, fmt.Sprintf("couldn't parse id: %s", parts[2]))

		} else {
			err := p.deleteReminder(id)
			if err == nil {
				p.Bot.SendMessage(channel, fmt.Sprintf("successfully canceled reminder: %s", parts[2]))
			} else {
				p.Bot.SendMessage(channel, fmt.Sprintf("failed to find and cancel reminder: %s", parts[2]))
			}
		}
		return true
	}

	return false
}

func (p *ReminderPlugin) Help(channel string, parts []string) {
	p.Bot.SendMessage(channel, "Pester someone with a reminder. Try \"remind <user> in <duration> message\".\n\nUnsure about duration syntax? Check https://golang.org/pkg/time/#ParseDuration")
}

func (p *ReminderPlugin) Event(kind string, message msg.Message) bool {
	return false
}

func (p *ReminderPlugin) BotMessage(message msg.Message) bool {
	return false
}

func (p *ReminderPlugin) RegisterWeb() *string {
	return nil
}

func (p *ReminderPlugin) getNextReminder() *Reminder {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	rows, err := p.db.Query("select id, fromWho, toWho, what, remindWhen, channel from reminders order by remindWhen asc limit 1;")
	if err != nil {
		log.Print(err)
		return nil
	}
	defer rows.Close()

	once := false
	var reminder *Reminder
	for rows.Next() {
		if once {
			log.Print("somehow got multiple rows")
		}
		reminder = &Reminder{}

		var when string
		err := rows.Scan(&reminder.id, &reminder.from, &reminder.who, &reminder.what, &when, &reminder.channel)
		if err != nil {
			log.Print(err)
			return nil
		}
		reminder.when, err = time.Parse(TIMESTAMP, when)
		if err != nil {
			log.Print(err)
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
		log.Print(err)
	}
	return err
}

func (p *ReminderPlugin) deleteReminder(id int64) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	res, err := p.db.Exec(`delete from reminders where id = ?;`, id)
	if err != nil {
		log.Print(err)
	} else {
		if affected, err := res.RowsAffected(); err != nil {
			return err
		} else if affected != 1 {
			return errors.New("didn't delete any rows")
		}
	}
	return err
}

func (p *ReminderPlugin) getRemindersFormatted(queryString string) (string, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	rows, err := p.db.Query(queryString)
	if err != nil {
		log.Print(err)
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
	if counter == 1 {
		return "no pending reminders", nil
	}

	return reminders, nil
}

func (p *ReminderPlugin) getAllRemindersFormatted(channel string) (string, error) {
	return p.getRemindersFormatted("select id, fromWho, toWho, what, remindWhen from reminders order by remindWhen asc;")
}

func (p *ReminderPlugin) getAllRemindersFromMeFormatted(channel, me string) (string, error) {
	return p.getRemindersFormatted(fmt.Sprintf("select id, fromWho, toWho, what, remindWhen from reminders where fromWho = '%s' order by remindWhen asc;", me))
}

func (p *ReminderPlugin) getAllRemindersToMeFormatted(channel, me string) (string, error) {
	return p.getRemindersFormatted(fmt.Sprintf("select id, fromWho, toWho, what, remindWhen from reminders where toWho = '%s' order by remindWhen asc;", me))
}

func (p *ReminderPlugin) queueUpNextReminder() {
	nextReminder := p.getNextReminder()

	if nextReminder != nil {
		p.timer.Reset(nextReminder.when.Sub(time.Now().UTC()))
	}
}

func reminderer(p *ReminderPlugin) {
	for {
		<-p.timer.C

		reminder := p.getNextReminder()

		if reminder != nil && time.Now().UTC().After(reminder.when) {
			var message string
			if reminder.from == reminder.who {
				reminder.from = "you"
				message = fmt.Sprintf("Hey %s, you wanted to be reminded: %s", reminder.who, reminder.what)
			} else {
				message = fmt.Sprintf("Hey %s, %s wanted you to be reminded: %s", reminder.who, reminder.from, reminder.what)
			}

			p.Bot.SendMessage(reminder.channel, message)

			if err := p.deleteReminder(reminder.id); err != nil {
				log.Print(reminder.id)
				log.Print(err)
				log.Fatal("this will cause problems, we need to stop now.")
			}
		}

		p.queueUpNextReminder()
	}
}

func (p *ReminderPlugin) ReplyMessage(message msg.Message, identifier string) bool { return false }
