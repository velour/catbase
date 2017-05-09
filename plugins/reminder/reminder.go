// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package reminder

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/config"
)

type ReminderPlugin struct {
	Bot            bot.Bot
	reminders      []*Reminder
	mutex          *sync.Mutex
	timer          *time.Timer
	config         *config.Config
	nextReminderId int
}

type Reminder struct {
	id      int
	from    string
	who     string
	what    string
	when    time.Time
	channel string
}

type reminderSlice []*Reminder

func (s reminderSlice) Len() int {
	return len(s)
}

func (s reminderSlice) Less(i, j int) bool {
	return s[i].when.Before(s[j].when)
}

func (s reminderSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func New(bot bot.Bot) *ReminderPlugin {
	dur, _ := time.ParseDuration("1h")
	timer := time.NewTimer(dur)
	timer.Stop()

	plugin := &ReminderPlugin{
		Bot:            bot,
		reminders:      []*Reminder{},
		mutex:          &sync.Mutex{},
		timer:          timer,
		config:         bot.Config(),
		nextReminderId: 0,
	}
	go reminderer(plugin)

	return plugin
}

func reminderer(p *ReminderPlugin) {
	//welcome to the reminderererererererererer
	for {
		<-p.timer.C

		p.mutex.Lock()

		reminder := p.reminders[0]
		if len(p.reminders) >= 2 {
			p.reminders = p.reminders[1:]
			p.timer.Reset(p.reminders[0].when.Sub(time.Now()))
		} else {
			p.reminders = []*Reminder{}
		}

		p.mutex.Unlock()

		if reminder.from == reminder.who {
			reminder.from = "you"
		}

		message := fmt.Sprintf("Hey %s, %s wanted you to be reminded: %s", reminder.who, reminder.from, reminder.what)
		p.Bot.SendMessage(reminder.channel, message)
	}

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

			reminders := []*Reminder{}

			operator := strings.ToLower(parts[2])

			doConfirm := true

			if operator == "in" {
				//one off reminder
				//remind who in dur blah
				when := time.Now().Add(dur)
				what := strings.Join(parts[4:], " ")

				id := p.nextReminderId
				p.nextReminderId++

				reminders = append(reminders, &Reminder{
					id:      id,
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

				when := time.Now().Add(dur)
				endTime := time.Now().Add(dur2)
				what := strings.Join(parts[6:], " ")

				for i := 0; when.Before(endTime); i++ {
					if i >= p.config.Reminder.MaxBatchAdd {
						p.Bot.SendMessage(channel, "Easy cowboy, that's a lot of reminders. I'll add some of them.")
						doConfirm = false
						break
					}

					id := p.nextReminderId
					p.nextReminderId++

					reminders = append(reminders, &Reminder{
						id:      id,
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

			if doConfirm {
				response := fmt.Sprintf("Sure %s, I'll remind %s.", from, who)
				p.Bot.SendMessage(channel, response)
			}

			p.mutex.Lock()

			p.timer.Stop()

			p.reminders = append(p.reminders, reminders...)

			sort.Sort(reminderSlice(p.reminders))

			if len(p.reminders) > 0 {
				p.timer.Reset(p.reminders[0].when.Sub(time.Now()))
			}

			p.mutex.Unlock()

			return true
		}
	} else if len(parts) == 2 && strings.ToLower(parts[0]) == "list" && strings.ToLower(parts[1]) == "reminders" {
		var response string
		p.mutex.Lock()
		if len(p.reminders) == 0 {
			response = "no pending reminders"
		} else {
			counter := 1
			for _, reminder := range p.reminders {
				if reminder.channel == channel {
					response += fmt.Sprintf("%d) %s -> %s :: %s @ %s (id=%d)\n", counter, reminder.from, reminder.who, reminder.what, reminder.when, reminder.id)
					counter++
				}
			}
		}
		p.mutex.Unlock()
		p.Bot.SendMessage(channel, response)
		return true
	} else if len(parts) == 3 && strings.ToLower(parts[0]) == "cancel" && strings.ToLower(parts[1]) == "reminder" {
		id, err := strconv.Atoi(parts[2])
		if err != nil {
			p.Bot.SendMessage(channel, fmt.Sprintf("couldn't parse id: %s", parts[2]))

		} else {
			p.mutex.Lock()
			deleted := false
			for i, reminder := range p.reminders {
				if reminder.id == id {
					copy(p.reminders[i:], p.reminders[i+1:])
					p.reminders[len(p.reminders)-1] = nil
					p.reminders = p.reminders[:len(p.reminders)-1]
					deleted = true
					break
				}
			}
			p.mutex.Unlock()

			if deleted {
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
