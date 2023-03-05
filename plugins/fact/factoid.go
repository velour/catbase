// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package fact

import (
	"database/sql"
	"fmt"
	"github.com/velour/catbase/config"
	"math/rand"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/jmoiron/sqlx"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
)

// The factoid plugin provides a learning system to the bot so that it can
// respond to queries in a way that is unpredictable and fun

// Factoid provides the necessary plugin-wide needs
type FactoidPlugin struct {
	b        bot.Bot
	c        *config.Config
	lastFact *Factoid
	db       *sqlx.DB
	handlers bot.HandlerTable
}

// NewFactoid creates a new Factoid with the Plugin interface
func New(botInst bot.Bot) *FactoidPlugin {
	p := &FactoidPlugin{
		b:  botInst,
		c:  botInst.Config(),
		db: botInst.DB(),
	}

	c := botInst.DefaultConnector()

	if _, err := p.db.Exec(`create table if not exists factoid (
			id integer primary key,
			fact string,
			tidbit string,
			verb string,
			owner string,
			created integer,
			accessed integer,
			count integer
		);`); err != nil {
		log.Fatal().Err(err)
	}

	if _, err := p.db.Exec(`create table if not exists factoid_alias (
			fact string,
			next string,
			primary key (fact, next)
		);`); err != nil {
		log.Fatal().Err(err)
	}

	for _, channel := range p.c.GetArray("channels", []string{}) {
		go p.factTimer(c, channel)

		go func(ch string) {
			// Some random time to start up
			time.Sleep(time.Duration(15) * time.Second)
			if ok, fact := p.findTrigger(p.c.Get("Factoid.StartupFact", "speed test")); ok {
				p.sayFact(c, msg.Message{
					Channel: ch,
					Body:    "speed test", // BUG: This is defined in the config too
					Command: true,
					Action:  false,
				}, *fact)
			}
		}(channel)
	}

	p.register()
	botInst.Register(p, bot.Help, p.help)

	p.registerWeb()

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
func (p *FactoidPlugin) learnFact(message msg.Message, fact, verb, tidbit string) error {
	verb = strings.ToLower(verb)
	if verb == "react" {
		// This would be a great place to check against the API for valid emojy
		// I'm too lazy for that
		tidbit = strings.Replace(tidbit, ":", "", -1)
		if len(strings.Split(tidbit, " ")) > 1 {
			return fmt.Errorf("That's not a valid emojy.")
		}
	}

	var count sql.NullInt64
	err := p.db.QueryRow(`select count(*) from factoid
		where fact=? and verb=? and tidbit=?`,
		fact, verb, tidbit).Scan(&count)
	if err != nil {
		log.Error().Err(err).Msg("Error counting facts")
		return fmt.Errorf("What?")
	} else if count.Valid && count.Int64 != 0 {
		log.Debug().Msg("User tried to relearn a fact.")
		return fmt.Errorf("Look, I already know that.")
	}

	n := Factoid{
		Fact:     fact,
		Tidbit:   tidbit,
		Verb:     verb,
		Owner:    message.User.Name,
		Created:  time.Now(),
		Accessed: time.Now(),
		Count:    0,
	}
	p.lastFact = &n
	err = n.Save(p.db)
	if err != nil {
		log.Error().Err(err).Msg("Error inserting fact")
		return fmt.Errorf("My brain is overheating.")
	}

	return nil
}

// findTrigger checks to see if a given string is a trigger or not
func (p *FactoidPlugin) findTrigger(fact string) (bool, *Factoid) {
	fact = strings.TrimSpace(strings.ToLower(fact))

	f, err := GetSingleFact(p.db, fact)
	if err != nil {
		log.Error().Err(err).Msg("GetSingleFact")
		return findAlias(p.db, fact)
	}
	return true, f
}

// sayFact spits out a fact to the channel and updates the fact in the database
// with new time and count information
func (p *FactoidPlugin) sayFact(c bot.Connector, message msg.Message, fact Factoid) {
	msg := p.b.Filter(message, fact.Tidbit)
	full := p.b.Filter(message, fmt.Sprintf("%s %s %s",
		fact.Fact, fact.Verb, fact.Tidbit,
	))
	for i, m := 0, strings.Split(msg, "$and"); i < len(m) && i < 4; i++ {
		msg := strings.TrimSpace(m[i])
		if len(msg) == 0 {
			continue
		}

		if fact.Verb == "action" {
			p.b.Send(c, bot.Action, message.Channel, msg)
		} else if fact.Verb == "react" {
			p.b.Send(c, bot.Reaction, message.Channel, msg, message)
		} else if fact.Verb == "reply" {
			p.b.Send(c, bot.Message, message.Channel, msg)
		} else if fact.Verb == "image" {
			p.sendImage(c, message, msg)
		} else {
			p.b.Send(c, bot.Message, message.Channel, full)
		}
	}

	// update fact tracking
	fact.Accessed = time.Now()
	fact.Count += 1
	err := fact.Save(p.db)
	if err != nil {
		log.Error().
			Interface("fact", fact).
			Err(err).
			Msg("could not update fact")
	}
	p.lastFact = &fact
}

func (p *FactoidPlugin) sendImage(c bot.Connector, message msg.Message, msg string) {
	imgSrc := ""
	txt := ""
	for _, w := range strings.Split(msg, " ") {
		if _, err := url.Parse(w); err == nil && strings.HasPrefix(w, "http") {
			log.Debug().Msgf("Valid image found: %s", w)
			imgSrc = w
		} else {
			txt = txt + " " + w
			log.Debug().Msgf("Adding %s to txt %s", w, txt)
		}
	}
	log.Debug().
		Str("imgSrc", imgSrc).
		Str("txt", txt).
		Str("msg", msg).
		Msg("Sending image attachment")
	if imgSrc != "" {
		if txt == "" {
			txt = imgSrc
		}
		img := bot.ImageAttachment{
			URL:    imgSrc,
			AltTxt: txt,
		}
		p.b.Send(c, bot.Message, message.Channel, "", img)
		return
	}
	p.b.Send(c, bot.Message, message.Channel, txt)
}

// trigger checks the message for its fitness to be a factoid and then hauls
// the message off to sayFact for processing if it is in fact a trigger
func (p *FactoidPlugin) trigger(c bot.Connector, message msg.Message) bool {
	minLen := p.c.GetInt("Factoid.MinLen", 4)
	if len(message.Body) > minLen || message.Command || message.Body == "..." {
		if ok, fact := p.findTrigger(message.Body); ok {
			p.sayFact(c, message, *fact)
			return true
		}
		r := strings.NewReplacer("'", "", "\"", "", ",", "", ".", "", ":", "",
			"?", "", "!", "")
		if ok, fact := p.findTrigger(r.Replace(message.Body)); ok {
			p.sayFact(c, message, *fact)
			return true
		}
	}

	return false
}

// tellThemWhatThatWas is a hilarious name for a function.
func (p *FactoidPlugin) tellThemWhatThatWas(c bot.Connector, message msg.Message) bool {
	fact := p.lastFact
	var msg string
	if fact == nil {
		msg = "Nope."
	} else {
		msg = fmt.Sprintf("That was (#%d) '%s <%s> %s'",
			fact.ID.Int64, fact.Fact, fact.Verb, fact.Tidbit)
	}
	p.b.Send(c, bot.Message, message.Channel, msg)
	return true
}

func (p *FactoidPlugin) learnAction(c bot.Connector, message msg.Message, action string) bool {
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
		p.b.Send(c, bot.Message, message.Channel, "I don't want to learn that.")
		return true
	}

	if len(strings.Split(fact, "$and")) > 4 {
		p.b.Send(c, bot.Message, message.Channel, "You can't use more than 4 $and operators.")
		return true
	}

	strippedaction := strings.Replace(strings.Replace(action, "<", "", 1), ">", "", 1)

	if err := p.learnFact(message, trigger, strippedaction, fact); err != nil {
		p.b.Send(c, bot.Message, message.Channel, err.Error())
	} else {
		p.b.Send(c, bot.Message, message.Channel, fmt.Sprintf("Okay, %s.", message.User.Name))
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
func (p *FactoidPlugin) forgetLastFact(c bot.Connector, message msg.Message) bool {
	if p.lastFact == nil {
		p.b.Send(c, bot.Message, message.Channel, "I refuse.")
		return true
	}

	err := p.lastFact.delete(p.db)
	if err != nil {
		log.Error().
			Err(err).
			Interface("lastFact", p.lastFact).
			Msg("Error removing fact")
	}
	fmt.Printf("Forgot #%d: %s %s %s\n", p.lastFact.ID.Int64, p.lastFact.Fact,
		p.lastFact.Verb, p.lastFact.Tidbit)
	p.b.Send(c, bot.Action, message.Channel, "hits himself over the head with a skillet")
	p.lastFact = nil

	return true
}

// Allow users to change facts with a simple regexp
func (p *FactoidPlugin) changeFact(c bot.Connector, message msg.Message) bool {
	oper := changeOperator(message.Body)
	parts := strings.SplitN(message.Body, oper, 2)
	userexp := strings.TrimSpace(parts[1])
	trigger := strings.TrimSpace(parts[0])

	parts = strings.Split(userexp, "/")

	log.Debug().
		Str("trigger", trigger).
		Str("userexp", userexp).
		Strs("parts", parts).
		Msg("changefact")

	if len(parts) == 4 {
		// replacement
		if parts[0] != "s" {
			p.b.Send(c, bot.Message, message.Channel, "Nah.")
		}
		find := parts[1]
		replace := parts[2]

		// replacement
		result, err := getFacts(p.db, trigger, parts[1])
		if err != nil {
			log.Error().
				Err(err).
				Str("trigger", trigger).
				Msg("Error getting facts")
		}
		if userexp[len(userexp)-1] != 'g' {
			result = result[:1]
		}
		// make the changes
		msg := fmt.Sprintf("Changing %d facts.", len(result))
		p.b.Send(c, bot.Message, message.Channel, msg)
		reg, err := regexp.Compile(find)
		if err != nil {
			p.b.Send(c, bot.Message, message.Channel, "I don't really want to.")
			return false
		}
		for _, fact := range result {
			fact.Fact = reg.ReplaceAllString(fact.Fact, replace)
			fact.Fact = strings.ToLower(fact.Fact)
			fact.Verb = reg.ReplaceAllString(fact.Verb, replace)
			fact.Tidbit = reg.ReplaceAllString(fact.Tidbit, replace)
			fact.Count += 1
			fact.Accessed = time.Now()
			fact.Save(p.db)
		}
	} else if len(parts) == 3 {
		// search for a factoid and print it
		result, err := getFacts(p.db, trigger, parts[1])
		if err != nil {
			log.Error().
				Err(err).
				Str("trigger", trigger).
				Msg("Error getting facts")
			p.b.Send(c, bot.Message, message.Channel, "bzzzt")
			return true
		}
		count := len(result)
		if count == 0 {
			p.b.Send(c, bot.Message, message.Channel, "I didn't find any facts like that.")
			return true
		}
		if parts[2] == "g" && len(result) > 4 {
			// summarize
			result = result[:4]
		} else {
			p.sayFact(c, message, *result[0])
			return true
		}
		msg := fmt.Sprintf("%s ", trigger)
		for i, fact := range result {
			if i != 0 {
				msg = fmt.Sprintf("%s |", msg)
			}
			msg = fmt.Sprintf("%s <%s> %s", msg, fact.Verb, fact.Tidbit)
		}
		if count > 4 {
			msg = fmt.Sprintf("%s | ...and %d others", msg, count)
		}
		p.b.Send(c, bot.Message, message.Channel, msg)
	} else {
		p.b.Send(c, bot.Message, message.Channel, "I don't know what you mean.")
	}
	return true
}

func (p *FactoidPlugin) register() {
	p.handlers = bot.HandlerTable{
		bot.HandlerSpec{Kind: bot.Message, IsCmd: true,
			Regex: regexp.MustCompile(`(?i)^what was that\??$`),
			Handler: func(r bot.Request) bool {
				return p.tellThemWhatThatWas(r.Conn, r.Msg)
			}},
		bot.HandlerSpec{Kind: bot.Message, IsCmd: true,
			Regex: regexp.MustCompile(`(?i)^alias (?P<from>\S+) to (?P<to>\S+)$`),
			Handler: func(r bot.Request) bool {
				from := r.Values["from"]
				to := r.Values["to"]
				log.Debug().Msgf("alias: %+v", r)
				a := aliasFromStrings(from, to)
				if err := a.save(p.db); err != nil {
					p.b.Send(r.Conn, bot.Message, r.Msg.Channel, err.Error())
				} else {
					p.b.Send(r.Conn, bot.Action, r.Msg.Channel, "learns a new synonym")
				}
				return true
			}},
		bot.HandlerSpec{Kind: bot.Message, IsCmd: true,
			Regex: regexp.MustCompile(`(?i)^factoid$`),
			Handler: func(r bot.Request) bool {
				fact := p.randomFact()
				p.sayFact(r.Conn, r.Msg, *fact)
				return true
			}},
		bot.HandlerSpec{Kind: bot.Message, IsCmd: true,
			Regex: regexp.MustCompile(`(?i)^forget that$`),
			Handler: func(r bot.Request) bool {
				return p.forgetLastFact(r.Conn, r.Msg)
			}},
		bot.HandlerSpec{Kind: bot.Message, IsCmd: false,
			Regex: regexp.MustCompile(`.*`),
			Handler: func(r bot.Request) bool {
				message := r.Msg
				c := r.Conn

				log.Debug().Msgf("Message: %+v", r)

				if !message.Command {
					// look for any triggers in the db matching this message
					return p.trigger(c, message)
				}

				if changeOperator(message.Body) != "" {
					return p.changeFact(c, message)
				}

				action := findAction(message.Body)
				if action != "" {
					return p.learnAction(c, message, action)
				}

				// look for any triggers in the db matching this message
				if p.trigger(c, message) {
					return true
				}

				return false
			}},
	}
	p.b.RegisterTable(p, p.handlers)
}

// Help responds to help requests. Every plugin must implement a help function.
func (p *FactoidPlugin) help(c bot.Connector, kind bot.Kind, message msg.Message, args ...any) bool {
	p.b.Send(c, bot.Message, message.Channel, "I can learn facts and spit them back out. You can say \"this is that\" or \"he <has> $5\". Later, trigger the factoid by just saying the trigger word, \"this\" or \"he\" in these examples.")
	p.b.Send(c, bot.Message, message.Channel, "I can also figure out some variables including: $nonzero, $digit, $nick, and $someone.")
	return true
}

// Pull a fact at random from the database
func (p *FactoidPlugin) randomFact() *Factoid {
	f, err := GetSingle(p.db)
	if err != nil {
		fmt.Println("Error getting a fact: ", err)
		return nil
	}
	return f
}

// factTimer spits out a fact at a given interval and with given probability
func (p *FactoidPlugin) factTimer(c bot.Connector, channel string) {
	quoteTime := p.c.GetInt("Factoid.QuoteTime", 30)
	if quoteTime == 0 {
		quoteTime = 30
		p.c.Set("Factoid.QuoteTime", "30")
	}
	duration := time.Duration(quoteTime) * time.Minute
	myLastMsg := time.Now()
	for {
		time.Sleep(time.Duration(5) * time.Second) // why 5? // no seriously, why 5?

		lastmsg, err := p.b.LastMessage(channel)
		if err != nil {
			// Probably no previous message to time off of
			continue
		}

		tdelta := time.Since(lastmsg.Time)
		earlier := time.Since(myLastMsg) > tdelta
		chance := rand.Float64()
		quoteChance := p.c.GetFloat64("Factoid.QuoteChance", 0.99)
		success := chance < quoteChance

		if success && tdelta > duration && earlier {
			fact := p.randomFact()
			if fact == nil {
				log.Debug().Msg("Didn't find a random fact to say")
				continue
			}

			users := p.b.Who(channel)

			// we need to fabricate a message so that bot.Filter can operate
			message := msg.Message{
				User:    &users[rand.Intn(len(users))],
				Channel: channel,
			}
			p.sayFact(c, message, *fact)
			myLastMsg = time.Now()
		}
	}
}
