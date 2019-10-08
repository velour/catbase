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
	b.Register(pp, bot.Message, pp.message)
	b.Register(pp, bot.Help, pp.help)
	return pp
}

// Message responds to the bot hook on recieving messages.
// This function returns true if the plugin responds in a meaningful way to the users message.
// Otherwise, the function returns false and the bot continues execution of other plugins.
func (p *PickerPlugin) message(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	if !strings.HasPrefix(message.Body, "pick") {
		return false
	}

	n, items, err := p.parse(message.Body)
	if err != nil {
		p.bot.Send(c, bot.Message, message.Channel, err.Error())
		return true
	}

	if n == 1 {
		item := items[rand.Intn(len(items))]
		out := fmt.Sprintf("I've chosen %q for you.", strings.TrimSpace(item))
		p.bot.Send(c, bot.Message, message.Channel, out)
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
	p.bot.Send(c, bot.Message, message.Channel, b.String())
	return true
}

var pickerListPrologue = regexp.MustCompile(`^pick([^ \t]*)[ \t]+([0-9]*)[ \t]*\{[ \t]*`)
var pickerListFinalItem = regexp.MustCompile(`^([^,}]+),?[ \t]*\}[ \t]*`)

func (p *PickerPlugin) parse(body string) (int, []string, error) {
	subs := pickerListPrologue.FindStringSubmatch(body)
	if subs == nil {
		return 0, nil, errors.New("saddle up for a syntax error")
	}

	log.Debug().
		Str("body", body).
		Interface("subs", subs).
		Msg("subs")

	n := 1
	delim := ","
	var err error

	if subs[1] != "" {
		delim = subs[1]
	}

	if subs[2] != "" {
		n, err = strconv.Atoi(subs[2])
		if err != nil {
			return 0, nil, err
		}
	}
	pickerListItem := regexp.MustCompile(`^([^` + delim + `]+)` + delim + `[ \t]*`)

	var items []string
	rest := body[len(subs[0]):]
	for {
		subs = pickerListItem.FindStringSubmatch(rest)
		if subs == nil {
			break
		}

		items = append(items, subs[1])
		rest = rest[len(subs[0]):]
	}

	subs = pickerListFinalItem.FindStringSubmatch(rest)
	if subs == nil {
		return 0, nil, errors.New("lasso yerself that final curly brace, compadre")
	}
	items = append(items, subs[1])

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
