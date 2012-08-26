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
	Bot  *bot.Bot
	Coll *mgo.Collection
}

// NewFactoidPlugin creates a new FactoidPlugin with the Plugin interface
func NewFactoidPlugin(bot *bot.Bot) *FactoidPlugin {
	p := &FactoidPlugin{
		Bot:  bot,
		Coll: nil,
	}
	p.LoadData()
	return p
}

func (p *FactoidPlugin) findAction(message string) string {
	r, err := regexp.Compile("<.*?>| is | are ")
	if err != nil {
		panic(err)
	}
	action := r.FindString(message)
	return action
}

func (p *FactoidPlugin) learnFact(message bot.Message, trigger, operator, fact string) {
	// if it's an action, we only want the fact part of it in the fulltext
	full := fact
	if operator != "action" && operator != "reply" {
		full = fmt.Sprintf("%s %s %s", trigger, operator, fact)
	}

	newfact := factoid{
		Id:        0,
		Trigger:   trigger,
		Operator:  operator,
		FullText:  full,
		CreatedBy: message.User.Name,
	}
	p.Coll.Insert(newfact)

}

func (p *FactoidPlugin) findTrigger(message string) (bool, string) {
	var results []factoid
	iter := p.Coll.Find(bson.M{"trigger": strings.ToLower(message)}).Iter()
	err := iter.All(&results)
	if err != nil {
		panic(err)
	}

	nfacts := len(results)
	if nfacts == 0 {
		return false, ""
	}

	fact := results[rand.Intn(nfacts)]
	return true, fact.FullText
}

// Message responds to the bot hook on recieving messages.
// This function returns true if the plugin responds in a meaningful way to the users message.
// Otherwise, the function returns false and the bot continues execution of other plugins.
func (p *FactoidPlugin) Message(message bot.Message) bool {
	// This bot does not reply to anything

	// This plugin has no business with normal messages
	if !message.Command {
		return false
	}

	action := p.findAction(message.Body)
	if action != "" {
		parts := strings.SplitN(message.Body, action, 2)
		// This could fail if is were the last word or it weren't in the sentence (like no spaces)
		if len(parts) != 2 {
			return false
		}

		trigger := strings.TrimSpace(parts[0])
		trigger = strings.ToLower(trigger)
		fact := strings.TrimSpace(parts[1])
		action := strings.TrimSpace(action)

		strippedaction := strings.Replace(strings.Replace(action, "<", "", 1), ">", "", 1)

		p.learnFact(message, trigger, strippedaction, fact)
		p.Bot.SendMessage(message.Channel, fmt.Sprintf("Okay, %s.", message.User.Name))

		return true
	}

	// look for any triggers in the db matching this message
	if ok, fact := p.findTrigger(message.Body); ok {
		fact = p.Bot.Filter(message, fact)
		p.Bot.SendMessage(message.Channel, fact)
		return true
	}

	p.Bot.SendMessage(message.Channel, "I don't know.")
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
