// © 2013 the AlePale Authors under the WTFPL. See AUTHORS for the list of authors.

package plugins

import (
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/chrissexton/alepale/bot"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

// The factoid plugin provides a learning system to the bot so that it can
// respond to queries in a way that is unpredictable and fun

// factoid stores info about our factoid for lookup and later interaction
type Factoid struct {
	Id           bson.ObjectId `bson:"_id,omitempty"`
	Idx          int
	Trigger      string
	Operator     string
	FullText     string
	Action       string
	CreatedBy    string
	DateCreated  time.Time
	LastAccessed time.Time
	Updated      time.Time
	AccessCount  int
}

// FactoidPlugin provides the necessary plugin-wide needs
type FactoidPlugin struct {
	Bot      *bot.Bot
	Coll     *mgo.Collection
	NotFound []string
	LastFact *Factoid
}

// NewFactoidPlugin creates a new FactoidPlugin with the Plugin interface
func NewFactoidPlugin(botInst *bot.Bot) *FactoidPlugin {
	p := &FactoidPlugin{
		Bot:  botInst,
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
	for _, channel := range botInst.Config.Channels {
		go p.factTimer(channel)

		go func() {
			// Some random time to start up
			time.Sleep(time.Duration(15) * time.Second)
			if ok, fact := p.findTrigger(p.Bot.Config.StartupFact); ok {
				p.sayFact(bot.Message{
					Channel: channel,
					Body:    "speed test",
					Command: true,
					Action:  false,
				}, *fact)
			}
		}()
	}

	return p
}

// findAction simply regexes a string for the action verb
func findAction(message string) string {
	r, err := regexp.Compile("<.+?>")
	if err != nil {
		panic(err)
	}
	action := r.FindString(message)

	if action == "" {
		if strings.Contains(message, " is ") {
			return "is"
		} else if strings.Contains(message, " are ") {
			return "are"
		}
	}

	return action
}

// learnFact assumes we have a learning situation and inserts a new fact
// into the database
func (p *FactoidPlugin) learnFact(message bot.Message, trigger, operator, fact string) bool {
	// if it's an action, we only want the fact part of it in the fulltext
	full := fact
	if operator != "action" && operator != "reply" {
		full = fmt.Sprintf("%s %s %s", trigger, operator, fact)
	}
	trigger = strings.ToLower(trigger)

	q := p.Coll.Find(bson.M{"trigger": trigger, "operator": operator, "fulltext": full})
	if n, _ := q.Count(); n != 0 {
		return false
	}

	// definite error here if no func setup
	// let's just aggregate
	var count map[string]interface{}
	query := []bson.M{{
		"$group": bson.M{
			"_id": nil,
			"idx": bson.M{
				"$max": "$idx",
			},
		},
	}}
	pipe := p.Coll.Pipe(query)
	err := pipe.One(&count)
	if err != nil {
		panic(err)
	}
	id := count["idx"].(int) + 1

	newfact := Factoid{
		Id:        bson.NewObjectId(),
		Idx:       id,
		Trigger:   trigger,
		Operator:  operator,
		FullText:  full,
		Action:    fact,
		CreatedBy: message.User.Name,
	}
	p.Coll.Insert(newfact)
	p.LastFact = &newfact
	return true
}

// findTrigger checks to see if a given string is a trigger or not
func (p *FactoidPlugin) findTrigger(message string) (bool, *Factoid) {
	var results []Factoid
	iter := p.Coll.Find(bson.M{"trigger": strings.ToLower(message)}).Iter()
	err := iter.All(&results)
	if err != nil {
		return false, nil
	}

	nfacts := len(results)
	if nfacts == 0 {
		return false, nil
	}

	fact := results[rand.Intn(nfacts)]
	return true, &fact
}

// sayFact spits out a fact to the channel and updates the fact in the database
// with new time and count information
func (p *FactoidPlugin) sayFact(message bot.Message, fact Factoid) {
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

	// update fact tracking
	fact.LastAccessed = time.Now()
	fact.AccessCount += 1
	err := p.Coll.UpdateId(fact.Id, fact)
	if err != nil {
		fmt.Printf("Could not update fact.\n")
		fmt.Printf("%#v\n", fact)
	}
	p.LastFact = &fact
}

// trigger checks the message for its fitness to be a factoid and then hauls
// the message off to sayFact for processing if it is in fact a trigger
func (p *FactoidPlugin) trigger(message bot.Message) bool {
	if len(message.Body) > 4 || message.Command || message.Body == "..." {
		if ok, fact := p.findTrigger(message.Body); ok {
			p.sayFact(message, *fact)
			return true
		}
		r := strings.NewReplacer("'", "", "\"", "", ",", "", ".", "", ":", "",
			"?", "", "!", "")
		if ok, fact := p.findTrigger(r.Replace(message.Body)); ok {
			p.sayFact(message, *fact)
			return true
		}
	}

	return false
}

// tellThemWhatThatWas is a hilarious name for a function.
func (p *FactoidPlugin) tellThemWhatThatWas(message bot.Message) bool {
	fact := p.LastFact
	var msg string
	if fact == nil {
		msg = "Nope."
	} else {
		msg = fmt.Sprintf("That was (#%d) '%s <%s> %s'",
			fact.Idx, fact.Trigger, fact.Operator, fact.Action)
	}
	p.Bot.SendMessage(message.Channel, msg)
	return true
}

func (p *FactoidPlugin) learnAction(message bot.Message, action string) bool {
	body := message.Body

	parts := strings.SplitN(body, action, 2)
	// This could fail if is were the last word or it weren't in the sentence (like no spaces)
	if len(parts) != 2 {
		return false
	}

	trigger := strings.TrimSpace(parts[0])
	fact := strings.TrimSpace(parts[1])
	action = strings.TrimSpace(action)

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

// Checks body for the ~= operator returns it
func changeOperator(body string) string {
	if strings.Contains(body, "=~") {
		return "=~"
	} else if strings.Contains(body, "~=") {
		return "~="
	}
	return ""
}

// If the user requesting forget is either the owner of the last learned fact or
// an admin, it may be deleted
func (p *FactoidPlugin) forgetLastFact(message bot.Message) bool {
	if p.LastFact == nil {
		p.Bot.SendMessage(message.Channel, "I refuse.")
		return true
	}
	if message.User.Admin || message.User.Name == p.LastFact.CreatedBy {
		err := p.Coll.Remove(bson.M{"_id": p.LastFact.Id})
		if err != nil {
			panic(err)
		}
		fmt.Printf("Forgot #%d: %s %s %s\n", p.LastFact.Idx, p.LastFact.Trigger,
			p.LastFact.Operator, p.LastFact.Action)
		p.Bot.SendAction(message.Channel, "hits himself over the head with a skillet")
		p.LastFact = nil
	} else {
		p.Bot.SendMessage(message.Channel, "You don't own that fact.")
	}

	return true
}

// Allow users to change facts with a simple regexp
func (p *FactoidPlugin) changeFact(message bot.Message) bool {
	oper := changeOperator(message.Body)
	parts := strings.SplitN(message.Body, oper, 2)
	userexp := strings.TrimSpace(parts[1])
	trigger := strings.TrimSpace(parts[0])

	parts = strings.Split(userexp, "/")
	if len(parts) == 4 {
		// replacement
		if parts[0] != "s" {
			p.Bot.SendMessage(message.Channel, "Nah.")
		}
		find := parts[1]
		replace := parts[2]

		// replacement
		var result []Factoid
		iter := p.Coll.Find(bson.M{"trigger": trigger})
		if message.User.Admin && userexp[len(userexp)-1] == 'g' {
			iter.All(&result)
		} else {
			result = make([]Factoid, 1)
			iter.One(&result[0])
			if result[0].CreatedBy != message.User.Name && !message.User.Admin {
				p.Bot.SendMessage(message.Channel, "That's not your fact to edit.")
				return true
			}
		}
		// make the changes
		msg := fmt.Sprintf("Changing %d facts.", len(result))
		p.Bot.SendMessage(message.Channel, msg)
		reg, err := regexp.Compile(find)
		if err != nil {
			p.Bot.SendMessage(message.Channel, "I don't really want to.")
			return false
		}
		for _, fact := range result {
			fact.FullText = reg.ReplaceAllString(fact.FullText, replace)
			fact.Trigger = reg.ReplaceAllString(fact.Trigger, replace)
			fact.Trigger = strings.ToLower(fact.Trigger)
			fact.Operator = reg.ReplaceAllString(fact.Operator, replace)
			fact.Action = reg.ReplaceAllString(fact.Action, replace)
			fact.AccessCount += 1
			fact.LastAccessed = time.Now()
			fact.Updated = time.Now()
			p.Coll.UpdateId(fact.Id, fact)
		}
	} else if len(parts) == 3 {
		// search for a factoid and print it
		var result []Factoid
		iter := p.Coll.Find(bson.M{"trigger": trigger,
			"fulltext": bson.M{"$regex": parts[1]}})
		count, _ := iter.Count()
		if parts[2] == "g" {
			// summarize
			iter.Limit(4).All(&result)
		} else {
			result = make([]Factoid, 1)
			iter.One(&result[0])
			p.sayFact(message, result[0])
			return true
		}
		msg := fmt.Sprintf("%s ", trigger)
		for i, fact := range result {
			if i != 0 {
				msg = fmt.Sprintf("%s |", msg)
			}
			msg = fmt.Sprintf("%s <%s> %s", msg, fact.Operator, fact.Action)
		}
		if count > 4 {
			msg = fmt.Sprintf("%s | ...and %d others", msg, count)
		}
		p.Bot.SendMessage(message.Channel, msg)
	} else {
		p.Bot.SendMessage(message.Channel, "I don't know what you mean.")
	}
	return true
}

// Message responds to the bot hook on recieving messages.
// This function returns true if the plugin responds in a meaningful way to the users message.
// Otherwise, the function returns false and the bot continues execution of other plugins.
func (p *FactoidPlugin) Message(message bot.Message) bool {
	if strings.ToLower(message.Body) == "what was that?" {
		return p.tellThemWhatThatWas(message)
	}

	// This plugin has no business with normal messages
	if !message.Command {
		// look for any triggers in the db matching this message
		return p.trigger(message)
	}

	if strings.ToLower(message.Body) == "factoid" {
		if fact := p.randomFact(); fact != nil {
			p.sayFact(message, *fact)
			return true
		} else {
			log.Println("Got a nil fact.")
		}
	}

	if strings.ToLower(message.Body) == "forget that" {
		return p.forgetLastFact(message)
	}

	if changeOperator(message.Body) != "" {
		return p.changeFact(message)
	}

	action := findAction(message.Body)
	if action != "" {
		return p.learnAction(message, action)
	}

	// look for any triggers in the db matching this message
	if p.trigger(message) {
		return true
	}

	// We didn't find anything, panic!
	p.Bot.SendMessage(message.Channel, p.NotFound[rand.Intn(len(p.NotFound))])
	return true
}

// LoadData imports any configuration data into the plugin. This is not strictly necessary other
// than the fact that the Plugin interface demands it exist. This may be deprecated at a later
// date.
func (p *FactoidPlugin) LoadData() {
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

// Pull a fact at random from the database
func (p *FactoidPlugin) randomFact() *Factoid {
	var fact Factoid

	nFacts, err := p.Coll.Count()
	if err != nil {
		return nil
	}

	// Possible bug here with no db
	if err := p.Coll.Find(nil).Skip(rand.Intn(nFacts)).One(&fact); err != nil {
		log.Println("Couldn't get next...")
	}

	return &fact
}

// factTimer spits out a fact at a given interval and with given probability
func (p *FactoidPlugin) factTimer(channel string) {
	duration := time.Duration(p.Bot.Config.QuoteTime) * time.Minute
	myLastMsg := time.Now()
	for {
		time.Sleep(time.Duration(5) * time.Second)

		lastmsg, err := p.Bot.LastMessage(channel)
		if err != nil {
			continue
		}

		tdelta := time.Since(lastmsg.Time)
		earlier := time.Since(myLastMsg) > tdelta
		chance := rand.Float64()
		success := chance < p.Bot.Config.QuoteChance

		if success && tdelta > duration && earlier {
			fact := p.randomFact()
			if fact == nil {
				continue
			}

			// we need to fabricate a message so that bot.Filter can operate
			message := bot.Message{
				User:    &p.Bot.Users[rand.Intn(len(p.Bot.Users))],
				Channel: channel,
			}
			p.sayFact(message, *fact)
			myLastMsg = time.Now()
		}
	}
}

// Handler for bot's own messages
func (p *FactoidPlugin) BotMessage(message bot.Message) bool {
	return false
}

// Register any web URLs desired
func (p *FactoidPlugin) RegisterWeb() *string {
	http.HandleFunc("/factoid/req", p.serveQuery)
	http.HandleFunc("/factoid", p.serveQuery)
	tmp := "/factoid"
	return &tmp
}

func linkify(text string) template.HTML {
	parts := strings.Split(text, " ")
	for i, word := range parts {
		if strings.HasPrefix(word, "http") {
			parts[i] = fmt.Sprintf("<a href=\"%s\">%s</a>", word, word)
		}
	}
	return template.HTML(strings.Join(parts, " "))
}

func (p *FactoidPlugin) serveQuery(w http.ResponseWriter, r *http.Request) {
	context := make(map[string]interface{})
	funcMap := template.FuncMap{
		// The name "title" is what the function will be called in the template text.
		"linkify": linkify,
	}
	if e := r.FormValue("entry"); e != "" {
		var entries []Factoid
		p.Coll.Find(bson.M{"trigger": bson.M{"$regex": e}}).All(&entries)
		context["Count"] = fmt.Sprintf("%d", len(entries))
		context["Entries"] = entries
		context["Search"] = e
	}
	t, err := template.New("factoidIndex").Funcs(funcMap).Parse(factoidIndex)
	if err != nil {
		log.Println(err)
	}
	t.Execute(w, context)
}
