package plugins

import (
	"bitbucket.org/phlyingpenguin/godeepintir/bot"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// This is a skeleton plugin to serve as an example and quick copy/paste for new plugins.

type BeersPlugin struct {
	Bot *bot.Bot
}

// NewBeersPlugin creates a new BeersPlugin with the Plugin interface
func NewBeersPlugin(bot *bot.Bot) *BeersPlugin {
	return &BeersPlugin{
		Bot: bot,
	}
}

type UserBeers struct {
	Nick      string
	berrcount int
	lastdrunk time.Time
	momentum  float64
}

// Message responds to the bot hook on recieving messages.
// This function returns true if the plugin responds in a meaningful way to the users message.
// Otherwise, the function returns false and the bot continues execution of other plugins.
func (p *BeersPlugin) Message(message bot.Message) bool {
	parts := strings.Fields(message.Body)

	if len(parts) == 0 {
		return false
	}

	channel := message.Channel
	user := message.User
	nick := user.Name

	// respond to the beers type of queries
	if parts[0] == "beers" {
		if len(parts) == 3 {

			// try to get a count out of parts[2]
			count, err := strconv.Atoi(parts[2])
			if err != nil {
				// if it's not a number, maybe it's a nick!
				if p.doIKnow(parts[2]) {
					p.reportCount(parts[2], false)
				} else {
					msg := fmt.Sprintf("Sorry, I don't know %s.", parts[2])
					p.Bot.SendMessage(channel, msg)
				}
			}

			if count < 0 {
				// you can't be negative
				msg := fmt.Sprintf("Sorry %s, you can't have negative beers!", nick)
				p.Bot.SendMessage(channel, msg)
			}
			if parts[1] == "+=" {
				p.setBeers(user, p.getBeers(user)+count)
			} else if parts[1] == "=" {
				if count == 0 {
					p.puke(user)
				} else {
					p.setBeers(user, count)
					p.reportCount(nick, true)
				}
			} else {
				p.Bot.SendMessage(channel, "I don't know your math.")
			}
		}

		// no matter what, if we're in here, then we've responded
		return true
	}

	if message.Command && parts[0] == "imbibe" {
		p.setBeers(user, p.getBeers(user)+1)
		p.reportCount(nick, true)
		return true
	}

	return false
}

// LoadData imports any configuration data into the plugin. This is not strictly necessary other
// than the fact that the Plugin interface demands it exist. This may be deprecated at a later
// date.
func (p *BeersPlugin) LoadData() {
	// This bot has no data to load
}

// Help responds to help requests. Every plugin must implement a help function.
func (p *BeersPlugin) Help(channel string, parts []string) {
	p.Bot.SendMessage(channel, "Sorry, Beers does not do a goddamn thing.")
}

func (p *BeersPlugin) setBeers(user *bot.User, amount int) {
}

func (p *BeersPlugin) addBeers(user *bot.User) {
}

func (p *BeersPlugin) getBeers(user *bot.User) int {
	return 0
}

func (p *BeersPlugin) hasBeers(user *bot.User) {
}

func (p *BeersPlugin) reportCount(user string, himself bool) {
}

func (p *BeersPlugin) puke(user *bot.User) {
}

func (p *BeersPlugin) doIKnow(user string) bool {
	return false
}
