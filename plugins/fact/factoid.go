// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package fact

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/velour/catbase/bot"
)

// The factoid plugin provides a learning system to the bot so that it can
// respond to queries in a way that is unpredictable and fun

// factoid stores info about our factoid for lookup and later interaction
type factoid struct {
	id       sql.NullInt64
	Fact     string
	Tidbit   string
	Verb     string
	Owner    string
	created  time.Time
	accessed time.Time
	Count    int
}

func (f *factoid) save(db *sqlx.DB) error {
	var err error
	if f.id.Valid {
		// update
		_, err = db.Exec(`update factoid set
			fact=?,
			tidbit=?,
			verb=?,
			owner=?,
			accessed=?,
			count=?
		where id=?`,
			f.Fact,
			f.Tidbit,
			f.Verb,
			f.Owner,
			f.accessed.Unix(),
			f.Count,
			f.id.Int64)
	} else {
		f.created = time.Now()
		f.accessed = time.Now()
		// insert
		res, err := db.Exec(`insert into factoid (
			fact,
			tidbit,
			verb,
			owner,
			created,
			accessed,
			count
		) values (?, ?, ?, ?, ?, ?, ?);`,
			f.Fact,
			f.Tidbit,
			f.Verb,
			f.Owner,
			f.created.Unix(),
			f.accessed.Unix(),
			f.Count,
		)
		if err != nil {
			return err
		}
		id, err := res.LastInsertId()
		// hackhackhack?
		f.id.Int64 = id
		f.id.Valid = true
	}
	return err
}

func (f *factoid) delete(db *sqlx.DB) error {
	var err error
	if f.id.Valid {
		_, err = db.Exec(`delete from factoid where id=?`, f.id)
	}
	f.id.Valid = false
	return err
}

func getFacts(db *sqlx.DB, fact string) ([]*factoid, error) {
	var fs []*factoid
	rows, err := db.Query(`select
			id,
			fact,
			tidbit,
			verb,
			owner,
			created,
			accessed,
			count
		from factoid
		where fact like ?;`,
		fmt.Sprintf("%%%s%%", fact))
	for rows.Next() {
		var f factoid
		var tmpCreated int64
		var tmpAccessed int64
		err := rows.Scan(
			&f.id,
			&f.Fact,
			&f.Tidbit,
			&f.Verb,
			&f.Owner,
			&tmpCreated,
			&tmpAccessed,
			&f.Count,
		)
		if err != nil {
			return nil, err
		}
		f.created = time.Unix(tmpCreated, 0)
		f.accessed = time.Unix(tmpAccessed, 0)
		fs = append(fs, &f)
	}
	return fs, err
}

func getSingle(db *sqlx.DB) (*factoid, error) {
	var f factoid
	var tmpCreated int64
	var tmpAccessed int64
	err := db.QueryRow(`select
			id,
			fact,
			tidbit,
			verb,
			owner,
			created,
			accessed,
			count
		from factoid
		order by random() limit 1;`).Scan(
		&f.id,
		&f.Fact,
		&f.Tidbit,
		&f.Verb,
		&f.Owner,
		&tmpCreated,
		&tmpAccessed,
		&f.Count,
	)
	f.created = time.Unix(tmpCreated, 0)
	f.accessed = time.Unix(tmpAccessed, 0)
	return &f, err
}

func getSingleFact(db *sqlx.DB, fact string) (*factoid, error) {
	var f factoid
	var tmpCreated int64
	var tmpAccessed int64
	err := db.QueryRow(`select
			id,
			fact,
			tidbit,
			verb,
			owner,
			created,
			accessed,
			count
		from factoid
		where fact like ?
		order by random() limit 1;`,
		fact).Scan(
		&f.id,
		&f.Fact,
		&f.Tidbit,
		&f.Verb,
		&f.Owner,
		&tmpCreated,
		&tmpAccessed,
		&f.Count,
	)
	f.created = time.Unix(tmpCreated, 0)
	f.accessed = time.Unix(tmpAccessed, 0)
	return &f, err
}

// FactoidPlugin provides the necessary plugin-wide needs
type FactoidPlugin struct {
	Bot      *bot.Bot
	NotFound []string
	LastFact *factoid
	db       *sqlx.DB
}

// NewFactoidPlugin creates a new FactoidPlugin with the Plugin interface
func NewFactoidPlugin(botInst *bot.Bot) *FactoidPlugin {
	p := &FactoidPlugin{
		Bot: botInst,
		NotFound: []string{
			"I don't know.",
			"NONONONO",
			"((",
			"*pukes*",
			"NOPE! NOPE! NOPE!",
			"One time, I learned how to jump rope.",
		},
		db: botInst.DB,
	}

	_, err := p.db.Exec(`create table if not exists factoid (
			id integer primary key,
			fact string,
			tidbit string,
			verb string,
			owner string,
			created integer,
			accessed integer,
			count integer
		);`)
	if err != nil {
		log.Fatal(err)
	}

	for _, channel := range botInst.Config.Channels {
		go p.factTimer(channel)

		go func(ch string) {
			// Some random time to start up
			time.Sleep(time.Duration(15) * time.Second)
			if ok, fact := p.findTrigger(p.Bot.Config.StartupFact); ok {
				p.sayFact(bot.Message{
					Channel: ch,
					Body:    "speed test", // BUG: This is defined in the config too
					Command: true,
					Action:  false,
				}, *fact)
			}
		}(channel)
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
func (p *FactoidPlugin) learnFact(message bot.Message, fact, verb, tidbit string) bool {
	verb = strings.ToLower(verb)

	var count sql.NullInt64
	err := p.db.QueryRow(`select count(*) from factoid
		where fact=? and verb=? and tidbit=?`,
		fact, verb, tidbit).Scan(&count)
	if err != nil {
		log.Println("Error counting facts: ", err)
		return false
	} else if count.Valid && count.Int64 != 0 {
		log.Println("User tried to relearn a fact.")
		return false
	}

	n := factoid{
		Fact:     fact,
		Tidbit:   tidbit,
		Verb:     verb,
		Owner:    message.User.Name,
		created:  time.Now(),
		accessed: time.Now(),
		Count:    0,
	}
	p.LastFact = &n
	err = n.save(p.db)
	if err != nil {
		log.Println("Error inserting fact: ", err)
		return false
	}

	return true
}

// findTrigger checks to see if a given string is a trigger or not
func (p *FactoidPlugin) findTrigger(fact string) (bool, *factoid) {
	fact = strings.ToLower(fact) // TODO: make sure this needs to be lowered here

	f, err := getSingleFact(p.db, fact)
	if err != nil {
		return false, nil
	}
	return true, f
}

// sayFact spits out a fact to the channel and updates the fact in the database
// with new time and count information
func (p *FactoidPlugin) sayFact(message bot.Message, fact factoid) {
	msg := p.Bot.Filter(message, fact.Tidbit)
	full := p.Bot.Filter(message, fmt.Sprintf("%s %s %s",
		fact.Fact, fact.Verb, fact.Tidbit,
	))
	for i, m := 0, strings.Split(msg, "$and"); i < len(m) && i < 4; i++ {
		msg := strings.TrimSpace(m[i])
		if len(msg) == 0 {
			continue
		}

		if fact.Verb == "action" {
			p.Bot.SendAction(message.Channel, msg)
		} else if fact.Verb == "reply" {
			p.Bot.SendMessage(message.Channel, msg)
		} else {
			p.Bot.SendMessage(message.Channel, full)
		}
	}

	// update fact tracking
	fact.accessed = time.Now()
	fact.Count += 1
	err := fact.save(p.db)
	if err != nil {
		log.Printf("Could not update fact.\n")
		log.Printf("%#v\n", fact)
		log.Println(err)
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
			fact.id.Int64, fact.Fact, fact.Verb, fact.Tidbit)
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
	if message.User.Admin || message.User.Name == p.LastFact.Owner {
		err := p.LastFact.delete(p.db)
		if err != nil {
			log.Println("Error removing fact: ", p.LastFact, err)
		}
		fmt.Printf("Forgot #%d: %s %s %s\n", p.LastFact.id.Int64, p.LastFact.Fact,
			p.LastFact.Verb, p.LastFact.Tidbit)
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
		result, err := getFacts(p.db, trigger)
		if err != nil {
			log.Println("Error getting facts: ", trigger, err)
		}
		if !(message.User.Admin && userexp[len(userexp)-1] == 'g') {
			result = result[:1]
			if result[0].Owner != message.User.Name && !message.User.Admin {
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
			fact.Fact = reg.ReplaceAllString(fact.Fact, replace)
			fact.Fact = strings.ToLower(fact.Fact)
			fact.Verb = reg.ReplaceAllString(fact.Verb, replace)
			fact.Tidbit = reg.ReplaceAllString(fact.Tidbit, replace)
			fact.Count += 1
			fact.accessed = time.Now()
			fact.save(p.db)
		}
	} else if len(parts) == 3 {
		// search for a factoid and print it
		result, err := getFacts(p.db, trigger)
		if err != nil {
			log.Println("Error getting facts: ", trigger, err)
		}
		count := len(result)
		if parts[2] == "g" {
			// summarize
			result = result[:4]
		} else {
			p.sayFact(message, *result[0])
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
func (p *FactoidPlugin) randomFact() *factoid {
	f, err := getSingle(p.db)
	if err != nil {
		fmt.Println("Error getting a fact: ", err)
		return nil
	}
	return f
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
		entries, err := getFacts(p.db, e)
		if err != nil {
			log.Println("Web error searching: ", err)
		}
		context["Count"] = fmt.Sprintf("%d", len(entries))
		context["Entries"] = entries
		context["Search"] = e
	}
	t, err := template.New("factoidIndex").Funcs(funcMap).Parse(factoidIndex)
	if err != nil {
		log.Println(err)
	}
	err = t.Execute(w, context)
	if err != nil {
		log.Println(err)
	}
}
