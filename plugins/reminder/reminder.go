// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package reminder

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
)

type ReminderPlugin struct {
	Bot       bot.Bot
	reminders []*Reminder
	mutex     *sync.Mutex
	timer     *time.Timer
}

type Reminder struct {
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
		Bot:       bot,
		reminders: []*Reminder{},
		mutex:     &sync.Mutex{},
		timer:     timer,
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
			dur, err := time.ParseDuration(parts[3])
			if err != nil {
				p.Bot.SendMessage(channel, "Easy cowboy, not sure I can parse that duration.")
				return true
			}
			when := time.Now().Add(dur)

			what := strings.Join(parts[4:], " ")

			response := fmt.Sprintf("Sure %s, I'll remind %s.", from, who)
			p.Bot.SendMessage(channel, response)

			p.mutex.Lock()

			p.timer.Stop()

			p.reminders = append(p.reminders, &Reminder{
				from:    from,
				who:     who,
				what:    what,
				when:    when,
				channel: channel,
			})

			sort.Sort(reminderSlice(p.reminders))

			p.timer.Reset(p.reminders[0].when.Sub(time.Now()))

			p.mutex.Unlock()

			return true
		}
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
