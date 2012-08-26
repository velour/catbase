package plugins

import (
	"bitbucket.org/phlyingpenguin/godeepintir/bot"
	"fmt"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"math/rand"
	"regexp"
	"strings"
	"time"
)

// This is a factoid plugin to serve as an example and quick copy/paste for new plugins.

// factoid stores info about our factoid for lookup and later interaction
type factoid struct {
	Id        int
	Trigger   string
	Operator  string
	FullText  string
	CreatedBy string
}

type FactoidPlugin struct {
	Bot      *bot.Bot
	Coll     *mgo.Collection
	NotFound []string
}

// NewFactoidPlugin creates a new FactoidPlugin with the Plugin interface
func NewFactoidPlugin(bot *bot.Bot) *FactoidPlugin {
	p := &FactoidPlugin{
		Bot:  bot,
		Coll: nil,
		NotFound: []string{
			"I don't know.",
			"NONONONO",
			"((",
			"*pukes*",
			"NOPE! NOPE! NOPE!",
			"One time, I learned how to jump rope.",
		},
	}
	p.LoadData()
	return p
}

func (p *FactoidPlugin) findAction(message string) string {
	r, err := regexp.Compile("<.+?>| is | are ")
	if err != nil {
		panic(err)
	}
	action := r.FindString(message)
	return action
}

func (p *FactoidPlugin) learnFact(message bot.Message, trigger, operator, fact string) bool {
	// if it's an action, we only want the fact part of it in the fulltext
	full := fact
	if operator != "action" && operator != "reply" {
		full = fmt.Sprintf("%s %s %s", trigger, operator, fact)
	}
	trigger = strings.ToLower(trigger)

	q := p.Coll.Find(bson.M{"trigger": trigger, "fulltext": full})
	if n, _ := q.Count(); n != 0 {
		return false
	}

	newfact := factoid{
		Id:        0,
		Trigger:   trigger,
		Operator:  operator,
		FullText:  full,
		CreatedBy: message.User.Name,
	}
	p.Coll.Insert(newfact)
	return true
}

func (p *FactoidPlugin) findTrigger(message string) (bool, *factoid) {
	var results []factoid
	iter := p.Coll.Find(bson.M{"trigger": strings.ToLower(message)}).Iter()
	err := iter.All(&results)
	if err != nil {
		panic(err)
	}

	nfacts := len(results)
	if nfacts == 0 {
		return false, nil
	}

	fact := results[rand.Intn(nfacts)]
	return true, &fact
}

func (p *FactoidPlugin) trigger(message bot.Message) bool {
	if len(message.Body) > 4 || message.Command {
		if ok, fact := p.findTrigger(message.Body); ok {
			msg := p.Bot.Filter(message, fact.FullText)
			for i, m := 0, strings.Split(msg, "$and"); i < len(m) && i < 4; i++ {
				msg := strings.TrimSpace(m[i])
				if len(msg) == 0 {
					continue
				}

				if fact.Operator == "action" {
					p.Bot.SendAction(message.Channel, msg)
				} else {
					p.Bot.SendMessage(message.Channel, msg)
				}
			}
			return true
		}
	}

	return false
}

// Message responds to the bot hook on recieving messages.
// This function returns true if the plugin responds in a meaningful way to the users message.
// Otherwise, the function returns false and the bot continues execution of other plugins.
func (p *FactoidPlugin) Message(message bot.Message) bool {
	// This bot does not reply to anything
	body := strings.TrimSpace(message.Body)

	// This plugin has no business with normal messages
	if !message.Command {
		// look for any triggers in the db matching this message
		return p.trigger(message)
	}

	action := p.findAction(body)
	if action != "" {
		parts := strings.SplitN(body, action, 2)
		// This could fail if is were the last word or it weren't in the sentence (like no spaces)
		if len(parts) != 2 {
			return false
		}

		trigger := strings.TrimSpace(parts[0])
		fact := strings.TrimSpace(parts[1])
		action := strings.TrimSpace(action)

		if len(trigger) == 0 || len(fact) == 0 || len(action) == 0 {
			p.Bot.SendMessage(message.Channel, "I don't want to learn that.")
			return true
		}

		if len(strings.Split(fact, "$and")) > 4 {
			p.Bot.SendMessage(message.Channel, "You can't use more than 4 $and operators.")
			return true
		}

		strippedaction := strings.Replace(strings.Replace(action, "<", "", 1), ">", "", 1)

		if p.learnFact(message, trigger, strippedaction, fact) {
			p.Bot.SendMessage(message.Channel, fmt.Sprintf("Okay, %s.", message.User.Name))
		} else {
			p.Bot.SendMessage(message.Channel, "I already know that.")
		}

		return true
	}

	// look for any triggers in the db matching this message
	if p.trigger(message) {
		return true
	}

	p.Bot.SendMessage(message.Channel, p.NotFound[rand.Intn(len(p.NotFound))])
	return true
}

// LoadData imports any configuration data into the plugin. This is not strictly necessary other
// than the fact that the Plugin interface demands it exist. This may be deprecated at a later
// date.
func (p *FactoidPlugin) LoadData() {
	// This bot has no data to load
	p.Coll = p.Bot.Db.C("factoid")
	rand.Seed(time.Now().Unix())
}

// Help responds to help requests. Every plugin must implement a help function.
func (p *FactoidPlugin) Help(channel string, parts []string) {
	p.Bot.SendMessage(channel, "I can learn facts and spit them back out. You can say \"this is that\" or \"he <has> $5\". Later, trigger the factoid by just saying the trigger word, \"this\" or \"he\" in these examples.")
	p.Bot.SendMessage(channel, "I can also figure out some variables including: $nonzero, $digit, $nick, and $someone.")
}

// Empty event handler because this plugin does not do anything on event recv
func (p *FactoidPlugin) Event(kind string, message bot.Message) bool {
	return false
}
