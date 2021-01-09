// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package fact

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"math/rand"
	"net/http"
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

// factoid stores info about our factoid for lookup and later interaction
type Factoid struct {
	ID       sql.NullInt64
	Fact     string
	Tidbit   string
	Verb     string
	Owner    string
	Created  time.Time
	Accessed time.Time
	Count    int
}

type alias struct {
	Fact string
	Next string
}

func (a *alias) resolve(db *sqlx.DB) (*Factoid, error) {
	// perform DB query to fill the To field
	q := `select fact, next from factoid_alias where fact=?`
	var next alias
	err := db.Get(&next, q, a.Next)
	if err != nil {
		// we hit the end of the chain, get a factoid named Next
		fact, err := GetSingleFact(db, a.Next)
		if err != nil {
			err := fmt.Errorf("Error resolvig alias %v: %v", a, err)
			return nil, err
		}
		return fact, nil
	}
	return next.resolve(db)
}

func findAlias(db *sqlx.DB, fact string) (bool, *Factoid) {
	q := `select * from factoid_alias where fact=?`
	var a alias
	err := db.Get(&a, q, fact)
	if err != nil {
		return false, nil
	}
	f, err := a.resolve(db)
	return err == nil, f
}

func (a *alias) save(db *sqlx.DB) error {
	q := `select * from factoid_alias where fact=?`
	var offender alias
	err := db.Get(&offender, q, a.Next)
	if err == nil {
		return fmt.Errorf("DANGER: an opposite alias already exists")
	}
	_, err = a.resolve(db)
	if err != nil {
		return fmt.Errorf("there is no fact at that destination")
	}
	q = `insert or replace into factoid_alias (fact, next) values (?, ?)`
	_, err = db.Exec(q, a.Fact, a.Next)
	if err != nil {
		return err
	}
	return nil
}

func aliasFromStrings(from, to string) *alias {
	return &alias{from, to}
}

func (f *Factoid) Save(db *sqlx.DB) error {
	var err error
	if f.ID.Valid {
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
			f.Accessed.Unix(),
			f.Count,
			f.ID.Int64)
	} else {
		f.Created = time.Now()
		f.Accessed = time.Now()
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
			f.Created.Unix(),
			f.Accessed.Unix(),
			f.Count,
		)
		if err != nil {
			return err
		}
		id, err := res.LastInsertId()
		// hackhackhack?
		f.ID.Int64 = id
		f.ID.Valid = true
	}
	return err
}

func (f *Factoid) delete(db *sqlx.DB) error {
	var err error
	if f.ID.Valid {
		_, err = db.Exec(`delete from factoid where id=?`, f.ID)
	}
	f.ID.Valid = false
	return err
}

func getFacts(db *sqlx.DB, fact string, tidbit string) ([]*Factoid, error) {
	var fs []*Factoid
	query := `select
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
		and tidbit like ?;`
	rows, err := db.Query(query,
		"%"+fact+"%", "%"+tidbit+"%")
	if err != nil {
		log.Error().Err(err).Msg("Error regexping for facts")
		return nil, err
	}
	for rows.Next() {
		var f Factoid
		var tmpCreated int64
		var tmpAccessed int64
		err := rows.Scan(
			&f.ID,
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
		f.Created = time.Unix(tmpCreated, 0)
		f.Accessed = time.Unix(tmpAccessed, 0)
		fs = append(fs, &f)
	}
	return fs, err
}

func GetSingle(db *sqlx.DB) (*Factoid, error) {
	var f Factoid
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
		&f.ID,
		&f.Fact,
		&f.Tidbit,
		&f.Verb,
		&f.Owner,
		&tmpCreated,
		&tmpAccessed,
		&f.Count,
	)
	f.Created = time.Unix(tmpCreated, 0)
	f.Accessed = time.Unix(tmpAccessed, 0)
	return &f, err
}

func GetSingleFact(db *sqlx.DB, fact string) (*Factoid, error) {
	var f Factoid
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
		&f.ID,
		&f.Fact,
		&f.Tidbit,
		&f.Verb,
		&f.Owner,
		&tmpCreated,
		&tmpAccessed,
		&f.Count,
	)
	f.Created = time.Unix(tmpCreated, 0)
	f.Accessed = time.Unix(tmpAccessed, 0)
	return &f, err
}

// Factoid provides the necessary plugin-wide needs
type FactoidPlugin struct {
	Bot      bot.Bot
	NotFound []string
	LastFact *Factoid
	db       *sqlx.DB
}

// NewFactoid creates a new Factoid with the Plugin interface
func New(botInst bot.Bot) *FactoidPlugin {
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

	for _, channel := range botInst.Config().GetArray("channels", []string{}) {
		go p.factTimer(c, channel)

		go func(ch string) {
			// Some random time to start up
			time.Sleep(time.Duration(15) * time.Second)
			if ok, fact := p.findTrigger(p.Bot.Config().Get("Factoid.StartupFact", "speed test")); ok {
				p.sayFact(c, msg.Message{
					Channel: ch,
					Body:    "speed test", // BUG: This is defined in the config too
					Command: true,
					Action:  false,
				}, *fact)
			}
		}(channel)
	}

	botInst.Register(p, bot.Message, p.message)
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
	p.LastFact = &n
	err = n.Save(p.db)
	if err != nil {
		log.Error().Err(err).Msg("Error inserting fact")
		return fmt.Errorf("My brain is overheating.")
	}

	return nil
}

// findTrigger checks to see if a given string is a trigger or not
func (p *FactoidPlugin) findTrigger(fact string) (bool, *Factoid) {
	fact = strings.ToLower(fact) // TODO: make sure this needs to be lowered here

	f, err := GetSingleFact(p.db, fact)
	if err != nil {
		return findAlias(p.db, fact)
	}
	return true, f
}

// sayFact spits out a fact to the channel and updates the fact in the database
// with new time and count information
func (p *FactoidPlugin) sayFact(c bot.Connector, message msg.Message, fact Factoid) {
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
			p.Bot.Send(c, bot.Action, message.Channel, msg)
		} else if fact.Verb == "react" {
			p.Bot.Send(c, bot.Reaction, message.Channel, msg, message)
		} else if fact.Verb == "reply" {
			p.Bot.Send(c, bot.Message, message.Channel, msg)
		} else if fact.Verb == "image" {
			p.sendImage(c, message, msg)
		} else {
			p.Bot.Send(c, bot.Message, message.Channel, full)
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
	p.LastFact = &fact
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
		p.Bot.Send(c, bot.Message, message.Channel, "", img)
		return
	}
	p.Bot.Send(c, bot.Message, message.Channel, txt)
}

// trigger checks the message for its fitness to be a factoid and then hauls
// the message off to sayFact for processing if it is in fact a trigger
func (p *FactoidPlugin) trigger(c bot.Connector, message msg.Message) bool {
	minLen := p.Bot.Config().GetInt("Factoid.MinLen", 4)
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
	fact := p.LastFact
	var msg string
	if fact == nil {
		msg = "Nope."
	} else {
		msg = fmt.Sprintf("That was (#%d) '%s <%s> %s'",
			fact.ID.Int64, fact.Fact, fact.Verb, fact.Tidbit)
	}
	p.Bot.Send(c, bot.Message, message.Channel, msg)
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
		p.Bot.Send(c, bot.Message, message.Channel, "I don't want to learn that.")
		return true
	}

	if len(strings.Split(fact, "$and")) > 4 {
		p.Bot.Send(c, bot.Message, message.Channel, "You can't use more than 4 $and operators.")
		return true
	}

	strippedaction := strings.Replace(strings.Replace(action, "<", "", 1), ">", "", 1)

	if err := p.learnFact(message, trigger, strippedaction, fact); err != nil {
		p.Bot.Send(c, bot.Message, message.Channel, err.Error())
	} else {
		p.Bot.Send(c, bot.Message, message.Channel, fmt.Sprintf("Okay, %s.", message.User.Name))
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
	if p.LastFact == nil {
		p.Bot.Send(c, bot.Message, message.Channel, "I refuse.")
		return true
	}

	err := p.LastFact.delete(p.db)
	if err != nil {
		log.Error().
			Err(err).
			Interface("LastFact", p.LastFact).
			Msg("Error removing fact")
	}
	fmt.Printf("Forgot #%d: %s %s %s\n", p.LastFact.ID.Int64, p.LastFact.Fact,
		p.LastFact.Verb, p.LastFact.Tidbit)
	p.Bot.Send(c, bot.Action, message.Channel, "hits himself over the head with a skillet")
	p.LastFact = nil

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
			p.Bot.Send(c, bot.Message, message.Channel, "Nah.")
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
		p.Bot.Send(c, bot.Message, message.Channel, msg)
		reg, err := regexp.Compile(find)
		if err != nil {
			p.Bot.Send(c, bot.Message, message.Channel, "I don't really want to.")
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
			p.Bot.Send(c, bot.Message, message.Channel, "bzzzt")
			return true
		}
		count := len(result)
		if count == 0 {
			p.Bot.Send(c, bot.Message, message.Channel, "I didn't find any facts like that.")
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
		p.Bot.Send(c, bot.Message, message.Channel, msg)
	} else {
		p.Bot.Send(c, bot.Message, message.Channel, "I don't know what you mean.")
	}
	return true
}

// Message responds to the bot hook on recieving messages.
// This function returns true if the plugin responds in a meaningful way to the users message.
// Otherwise, the function returns false and the bot continues execution of other plugins.
func (p *FactoidPlugin) message(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	if strings.ToLower(message.Body) == "what was that?" {
		return p.tellThemWhatThatWas(c, message)
	}

	// This plugin has no business with normal messages
	if !message.Command {
		// look for any triggers in the db matching this message
		return p.trigger(c, message)
	}

	if strings.HasPrefix(strings.ToLower(message.Body), "alias") {
		log.Debug().
			Str("alias", message.Body).
			Msg("Trying to learn an alias")
		m := strings.TrimPrefix(message.Body, "alias ")
		parts := strings.SplitN(m, "->", 2)
		if len(parts) != 2 {
			p.Bot.Send(c, bot.Message, message.Channel, "If you want to alias something, use: `alias this -> that`")
			return true
		}
		a := aliasFromStrings(strings.TrimSpace(parts[1]), strings.TrimSpace(parts[0]))
		if err := a.save(p.db); err != nil {
			p.Bot.Send(c, bot.Message, message.Channel, err.Error())
		} else {
			p.Bot.Send(c, bot.Action, message.Channel, "learns a new synonym")
		}
		return true
	}

	if strings.ToLower(message.Body) == "factoid" {
		if fact := p.randomFact(); fact != nil {
			p.sayFact(c, message, *fact)
			return true
		}
		log.Debug().Msg("Got a nil fact.")
	}

	if strings.ToLower(message.Body) == "forget that" {
		return p.forgetLastFact(c, message)
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

	// We didn't find anything, panic!
	p.Bot.Send(c, bot.Message, message.Channel, p.NotFound[rand.Intn(len(p.NotFound))])
	return true
}

// Help responds to help requests. Every plugin must implement a help function.
func (p *FactoidPlugin) help(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	p.Bot.Send(c, bot.Message, message.Channel, "I can learn facts and spit them back out. You can say \"this is that\" or \"he <has> $5\". Later, trigger the factoid by just saying the trigger word, \"this\" or \"he\" in these examples.")
	p.Bot.Send(c, bot.Message, message.Channel, "I can also figure out some variables including: $nonzero, $digit, $nick, and $someone.")
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
	quoteTime := p.Bot.Config().GetInt("Factoid.QuoteTime", 30)
	if quoteTime == 0 {
		quoteTime = 30
		p.Bot.Config().Set("Factoid.QuoteTime", "30")
	}
	duration := time.Duration(quoteTime) * time.Minute
	myLastMsg := time.Now()
	for {
		time.Sleep(time.Duration(5) * time.Second) // why 5?

		lastmsg, err := p.Bot.LastMessage(channel)
		if err != nil {
			// Probably no previous message to time off of
			continue
		}

		tdelta := time.Since(lastmsg.Time)
		earlier := time.Since(myLastMsg) > tdelta
		chance := rand.Float64()
		quoteChance := p.Bot.Config().GetFloat64("Factoid.QuoteChance", 0.99)
		if quoteChance == 0.0 {
			quoteChance = 0.99
			p.Bot.Config().Set("Factoid.QuoteChance", "0.99")
		}
		success := chance < quoteChance

		if success && tdelta > duration && earlier {
			fact := p.randomFact()
			if fact == nil {
				log.Debug().Msg("Didn't find a random fact to say")
				continue
			}

			users := p.Bot.Who(channel)

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

// Register any web URLs desired
func (p *FactoidPlugin) registerWeb() {
	http.HandleFunc("/factoid/api", p.serveAPI)
	http.HandleFunc("/factoid/req", p.serveQuery)
	http.HandleFunc("/factoid", p.serveQuery)
	p.Bot.RegisterWeb("/factoid", "Factoid")
}

func linkify(text string) template.HTML {
	parts := strings.Fields(text)
	for i, word := range parts {
		if strings.HasPrefix(word, "http") {
			parts[i] = fmt.Sprintf("<a href=\"%s\">%s</a>", word, word)
		}
	}
	return template.HTML(strings.Join(parts, " "))
}
func (p *FactoidPlugin) serveAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fmt.Fprintf(w, "Incorrect HTTP method")
		return
	}
	info := struct {
		Query string `json:"query"`
	}{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&info)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprint(w, err)
		return
	}

	entries, err := getFacts(p.db, info.Query, "")
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprint(w, err)
		return
	}

	data, err := json.Marshal(entries)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprint(w, err)
		return
	}
	w.Write(data)
}

func (p *FactoidPlugin) serveQuery(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, factoidIndex)
}
