// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package dice

import (
	"fmt"
	"math/rand"
	"regexp"
	"strconv"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
)

// This is a dice plugin to serve as an example and quick copy/paste for new plugins.

type DicePlugin struct {
	Bot bot.Bot
}

// New creates a new DicePlugin with the Plugin interface
func New(b bot.Bot) *DicePlugin {
	dp := &DicePlugin{
		Bot: b,
	}
	b.RegisterRegexCmd(dp, bot.Message, rollRegex, dp.rollCmd)
	b.Register(dp, bot.Help, dp.help)
	return dp
}

func rollDie(sides int) int {
	return rand.Intn(sides) + 1
}

var rollRegex = regexp.MustCompile(`^(?P<number>\d+)d(?P<sides>\d+)$`)

func (p *DicePlugin) rollCmd(r bot.Request) bool {
	nDice, _ := strconv.Atoi(r.Values["number"])
	sides, _ := strconv.Atoi(r.Values["sides"])

	if sides < 2 || nDice < 1 || nDice > 20 {
		p.Bot.Send(r.Conn, bot.Message, r.Msg.Channel, "You're a dick.")
		return true
	}

	rolls := fmt.Sprintf("%s, you rolled: ", r.Msg.User.Name)

	for i := 0; i < nDice; i++ {
		rolls = fmt.Sprintf("%s %d", rolls, rollDie(sides))
		if i != nDice-1 {
			rolls = fmt.Sprintf("%s,", rolls)
		} else {
			rolls = fmt.Sprintf("%s.", rolls)
		}
	}

	p.Bot.Send(r.Conn, bot.Message, r.Msg.Channel, rolls)
	return true
}

// Help responds to help requests. Every plugin must implement a help function.
func (p *DicePlugin) help(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	p.Bot.Send(c, bot.Message, message.Channel, "Roll dice using notation XdY. Try \"3d20\".")
	return true
}
