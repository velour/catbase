package tell

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/jmoiron/sqlx"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
)

type delayedMsg string

type TellPlugin struct {
	b  bot.Bot
	db *sqlx.DB
}

func New(b bot.Bot) *TellPlugin {
	tp := &TellPlugin{b, b.DB()}
	b.Register(tp, bot.Message, tp.message)
	tp.createDB()
	return tp
}

type tell struct {
	ID   int
	Who  string
	What string
}

func (t *TellPlugin) createDB() {
	q := `create table if not exists tell (
		id integer primary key autoincrement,
		who string,
		what string
	)`
	t.db.MustExec(q)
}

func (t *TellPlugin) getTells() []tell {
	result := []tell{}
	q := `select * from tell`
	t.db.Select(&result, q)
	return result
}

func (t *TellPlugin) rmTell(entry tell) {
	q := `delete from tell where id=?`
	if _, err := t.db.Exec(q, entry.ID); err != nil {
		log.Error().Err(err).Msg("could not remove tell")
	}
}

func (t *TellPlugin) addTell(who, what string) error {
	q := `insert into tell (who, what) values (?, ?)`
	_, err := t.db.Exec(q, who, what)
	if err != nil {
		log.Error().Err(err).Msg("could not add tell")
	}
	return err
}

func (t *TellPlugin) check(who string) []tell {
	result := []tell{}
	tells := t.getTells()
	for _, e := range tells {
		if e.Who == who {
			result = append(result, e)
			t.rmTell(e)
		}
	}
	return result
}

func (t *TellPlugin) checkValidTarget(ch, target string) bool {
	users := t.b.Who(ch)
	log.Debug().
		Str("ch", ch).
		Str("target", target).
		Interface("users", users).
		Msg("checking valid target")
	for _, u := range users {
		if u.Name == target {
			return true
		}
	}
	return false
}

func (t *TellPlugin) troll(who string) bool {
	targets := t.b.Config().GetArray("tell.troll", []string{})
	for _, target := range targets {
		if who == target {
			return true
		}
	}
	return false
}

func (t *TellPlugin) message(c bot.Connector, kind bot.Kind, message msg.Message, args ...any) bool {
	if strings.HasPrefix(strings.ToLower(message.Body), "tell ") ||
		strings.HasPrefix(strings.ToLower(message.Body), "tellah ") {
		parts := strings.Split(message.Body, " ")
		target := strings.ToLower(parts[1])
		if !t.checkValidTarget(message.Channel, target) {
			if t.troll(message.User.Name) {
				t.b.Send(c, bot.Message, message.Channel, fmt.Sprintf("Okay. I'll tell %s.", target))
				return true
			}
			return false
		}
		newMessage := strings.Join(parts[2:], " ")
		newMessage = fmt.Sprintf("Hey, %s. %s said: %s", target, message.User.Name, newMessage)
		t.addTell(target, newMessage)
		t.b.Send(c, bot.Message, message.Channel, fmt.Sprintf("Okay. I'll tell %s.", target))
		return true
	}
	uname := strings.ToLower(message.User.Name)
	if tells := t.check(uname); len(tells) > 0 {
		for _, m := range tells {
			t.b.Send(c, bot.Message, message.Channel, m.What)
		}
		return true
	}
	return false
}
