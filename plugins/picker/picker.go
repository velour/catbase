// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package picker

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"

	"fmt"
	"math/rand"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
)

type PickerPlugin struct {
	bot bot.Bot
}

// NewPickerPlugin creates a new PickerPlugin with the Plugin interface
func New(b bot.Bot) *PickerPlugin {
	pp := &PickerPlugin{
		bot: b,
	}
	b.RegisterRegex(pp, bot.Message, pickRegex, pp.message)
	b.Register(pp, bot.Help, pp.help)
	return pp
}

// Message responds to the bot hook on recieving messages.
// This function returns true if the plugin responds in a meaningful way to the users message.
// Otherwise, the function returns false and the bot continues execution of other plugins.
func (p *PickerPlugin) message(r bot.Request) bool {
	message := r.Msg

	n, items, err := p.parse(message.Body)
	if err != nil {
		p.bot.Send(r.Conn, bot.Message, message.Channel, err.Error())
		return true
	}

	preferences := p.bot.Config().GetArray("picker.preferences", []string{})
	for _, it := range items {
		for _, pref := range preferences {
			if pref == it {
				out := fmt.Sprintf("I've chosen %q for you.", strings.TrimSpace(it))
				p.bot.Send(r.Conn, bot.Message, message.Channel, out)
				return true
			}
		}
	}

	if n == 1 {
		item := items[rand.Intn(len(items))]
		out := fmt.Sprintf("I've chosen %q for you.", strings.TrimSpace(item))
		p.bot.Send(r.Conn, bot.Message, message.Channel, out)
		return true
	}

	rand.Shuffle(len(items), func(i, j int) {
		items[i], items[j] = items[j], items[i]
	})
	items = items[:n]

	var b strings.Builder
	b.WriteString("I've chosen these hot picks for you: { ")
	fmt.Fprintf(&b, "%q", items[0])
	for _, item := range items[1:] {
		fmt.Fprintf(&b, ", %q", item)
	}
	b.WriteString(" }")
	p.bot.Send(r.Conn, bot.Message, message.Channel, b.String())
	return true
}

var pickRegex = regexp.MustCompile(`^pick([^\s]+)?(\s\d)?\s+{(.*)}$`)

func (p *PickerPlugin) parse(body string) (int, []string, error) {
	subs := pickRegex.FindStringSubmatch(body)
	log.Debug().Str("body", body).Strs("subs", subs).Msg("parse")
	if subs == nil {
		return 0, nil, errors.New("saddle up for a syntax error")
	}

	log.Debug().
		Str("body", body).
		Interface("subs", subs).
		Msg("subs")

	delim := ","
	if subs[1] != "" {
		delim = subs[1]
	}

	n := 1
	var err error
	if num := strings.TrimSpace(subs[2]); num != "" {
		n, err = strconv.Atoi(num)
		if err != nil {
			return 0, nil, err
		}
	}

	items := strings.Split(subs[3], delim)

	if n < 1 || n > len(items) {
		return 0, nil, errors.New("whoah there, bucko, I can't create something out of nothing")
	}

	return n, items, nil
}

// Help responds to help requests. Every plugin must implement a help function.
func (p *PickerPlugin) help(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	p.bot.Send(c, bot.Message, message.Channel, "Choose from a list of options. Try \"pick {a,b,c}\".")
	return true
}
