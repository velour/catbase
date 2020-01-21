package counter

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/jmoiron/sqlx"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
)

// This is a counter plugin to count arbitrary things.

var teaMatcher = regexp.MustCompile("(?i)^([^.]+)\\. [^.]*\\. ([^.]*\\.?)+$")

type CounterPlugin struct {
	Bot bot.Bot
	DB  *sqlx.DB
}

type Item struct {
	*sqlx.DB

	ID    int64
	Nick  string
	Item  string
	Count int
}

type alias struct {
	*sqlx.DB

	ID       int64
	Item     string
	PointsTo string `db:"points_to"`
}

// GetItems returns all counters
func GetAllItems(db *sqlx.DB) ([]Item, error) {
	var items []Item
	err := db.Select(&items, `select * from counter`)
	if err != nil {
		return nil, err
	}
	// Don't forget to embed the DB into all of that shiz
	for i := range items {
		items[i].DB = db
	}
	return items, nil
}

// GetItems returns all counters for a subject
func GetItems(db *sqlx.DB, nick string) ([]Item, error) {
	var items []Item
	err := db.Select(&items, `select * from counter where nick = ?`, nick)
	if err != nil {
		return nil, err
	}
	// Don't forget to embed the DB into all of that shiz
	for i := range items {
		items[i].DB = db
	}
	return items, nil
}

func LeaderAll(db *sqlx.DB) ([]Item, error) {
	s := `select id,item,nick,max(count) as count from counter group by item having count(nick) > 1 and max(count) > 1 order by count desc`
	var items []Item
	err := db.Select(&items, s)
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].DB = db
	}
	return items, nil
}

func Leader(db *sqlx.DB, itemName string) ([]Item, error) {
	itemName = strings.ToLower(itemName)
	s := `select * from counter where item=? order by count desc`
	var items []Item
	err := db.Select(&items, s, itemName)
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].DB = db
	}
	return items, nil
}

func MkAlias(db *sqlx.DB, item, pointsTo string) (*alias, error) {
	item = strings.ToLower(item)
	pointsTo = strings.ToLower(pointsTo)
	res, err := db.Exec(`insert into counter_alias (item, points_to) values (?, ?)`,
		item, pointsTo)
	if err != nil {
		_, err := db.Exec(`update counter_alias set points_to=? where item=?`, pointsTo, item)
		if err != nil {
			return nil, err
		}
		var a alias
		if err := db.Get(&a, `select * from counter_alias where item=?`, item); err != nil {
			return nil, err
		}
		return &a, nil
	}
	id, _ := res.LastInsertId()

	return &alias{db, id, item, pointsTo}, nil
}

// GetItem returns a specific counter for a subject
func GetItem(db *sqlx.DB, nick, itemName string) (Item, error) {
	itemName = trimUnicode(itemName)
	var item Item
	item.DB = db
	var a alias
	if err := db.Get(&a, `select * from counter_alias where item=?`, itemName); err == nil {
		itemName = a.PointsTo
	} else {
		log.Error().Err(err).Interface("alias", a)
	}

	err := db.Get(&item, `select * from counter where nick = ? and item= ?`,
		nick, itemName)
	switch err {
	case sql.ErrNoRows:
		item.ID = -1
		item.Nick = nick
		item.Item = itemName
	case nil:
	default:
		return Item{}, err
	}
	log.Debug().
		Str("nick", nick).
		Str("itemName", itemName).
		Interface("item", item).
		Msg("got item")
	return item, nil
}

// Create saves a counter
func (i *Item) Create() error {
	res, err := i.Exec(`insert into counter (nick, item, count) values (?, ?, ?);`,
		i.Nick, i.Item, i.Count)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	// hackhackhack?
	i.ID = id
	return err
}

// UpdateDelta sets a value
// This will create or delete the item if necessary
func (i *Item) Update(value int) error {
	i.Count = value
	if i.Count == 0 && i.ID != -1 {
		return i.Delete()
	}
	if i.ID == -1 {
		i.Create()
	}
	log.Debug().
		Interface("i", i).
		Int("value", value).
		Msg("Updating item")
	_, err := i.Exec(`update counter set count = ? where id = ?`, i.Count, i.ID)
	return err
}

// UpdateDelta changes a value according to some delta
// This will create or delete the item if necessary
func (i *Item) UpdateDelta(delta int) error {
	i.Count += delta
	return i.Update(i.Count)
}

// Delete removes a counter from the database
func (i *Item) Delete() error {
	_, err := i.Exec(`delete from counter where id = ?`, i.ID)
	i.ID = -1
	return err
}

// NewCounterPlugin creates a new CounterPlugin with the Plugin interface
func New(b bot.Bot) *CounterPlugin {
	tx := b.DB().MustBegin()
	b.DB().MustExec(`create table if not exists counter (
			id integer primary key,
			nick string,
			item string,
			count integer
		);`)
	b.DB().MustExec(`create table if not exists counter_alias (
			id integer PRIMARY KEY AUTOINCREMENT,
			item string NOT NULL UNIQUE,
			points_to string NOT NULL
		);`)
	tx.Commit()
	cp := &CounterPlugin{
		Bot: b,
		DB:  b.DB(),
	}
	b.Register(cp, bot.Message, cp.message)
	b.Register(cp, bot.Help, cp.help)
	cp.registerWeb()
	return cp
}

func trimUnicode(s string) string {
	return strings.Trim(s, string(rune(0xFE0F)))
}

// Message responds to the bot hook on recieving messages.
// This function returns true if the plugin responds in a meaningful way to the
// users message. Otherwise, the function returns false and the bot continues
// execution of other plugins.
func (p *CounterPlugin) message(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	// This bot does not reply to anything
	nick := message.User.Name
	channel := message.Channel
	parts := strings.Fields(message.Body)

	if len(parts) == 0 {
		return false
	}

	if len(parts) == 3 && strings.ToLower(parts[0]) == "mkalias" {
		if _, err := MkAlias(p.DB, parts[1], parts[2]); err != nil {
			log.Error().Err(err)
			return false
		}
		p.Bot.Send(c, bot.Message, channel, fmt.Sprintf("Created alias %s -> %s",
			parts[1], parts[2]))
		return true
	} else if strings.ToLower(parts[0]) == "leaderboard" {
		var cmd func() ([]Item, error)
		itNameTxt := ""

		if len(parts) == 1 {
			cmd = func() ([]Item, error) { return LeaderAll(p.DB) }
		} else {
			itNameTxt = fmt.Sprintf(" for %s", parts[1])
			cmd = func() ([]Item, error) { return Leader(p.DB, parts[1]) }
		}

		its, err := cmd()
		if err != nil {
			log.Error().Err(err)
			return false
		} else if len(its) == 0 {
			return false
		}

		out := fmt.Sprintf("Leaderboard%s:\n", itNameTxt)
		for _, it := range its {
			out += fmt.Sprintf("%s with %d %s\n",
				it.Nick,
				it.Count,
				it.Item,
			)
		}
		p.Bot.Send(c, bot.Message, channel, out)
		return true
	} else if match := teaMatcher.MatchString(message.Body); match {
		// check for tea match TTT
		return p.checkMatch(c, message)
	} else if message.Command && message.Body == "reset me" {
		items, err := GetItems(p.DB, strings.ToLower(nick))
		if err != nil {
			log.Error().
				Err(err).
				Str("nick", nick).
				Msg("Error getting items to reset")
			p.Bot.Send(c, bot.Message, channel, "Something is technically wrong with your counters.")
			return true
		}
		log.Debug().Msgf("Items: %+v", items)
		for _, item := range items {
			item.Delete()
		}
		p.Bot.Send(c, bot.Message, channel, fmt.Sprintf("%s, you are as new, my son.", nick))
		return true
	} else if message.Command && parts[0] == "inspect" && len(parts) == 2 {
		var subject string

		if parts[1] == "me" {
			subject = strings.ToLower(nick)
		} else {
			subject = strings.ToLower(parts[1])
		}

		log.Debug().
			Str("subject", subject).
			Msg("Getting counter")
		// pull all of the items associated with "subject"
		items, err := GetItems(p.DB, subject)
		if err != nil {
			log.Error().
				Err(err).
				Str("subject", subject).
				Msg("Error retrieving items")
			p.Bot.Send(c, bot.Message, channel, "Something went wrong finding that counter;")
			return true
		}

		resp := fmt.Sprintf("%s has the following counters:", subject)
		count := 0
		for _, it := range items {
			count += 1
			if count > 1 {
				resp += ","
			}
			resp += fmt.Sprintf(" %s: %d", it.Item, it.Count)
			if count > 20 {
				resp += ", and a few others"
				break
			}
		}
		resp += "."

		if count == 0 {
			p.Bot.Send(c, bot.Message, channel, fmt.Sprintf("%s has no counters.", subject))
			return true
		}

		p.Bot.Send(c, bot.Message, channel, resp)
		return true
	} else if message.Command && len(parts) == 2 && parts[0] == "clear" {
		subject := strings.ToLower(nick)
		itemName := strings.ToLower(parts[1])

		it, err := GetItem(p.DB, subject, itemName)
		if err != nil {
			log.Error().
				Err(err).
				Str("subject", subject).
				Str("itemName", itemName).
				Msg("Error getting item to remove")
			p.Bot.Send(c, bot.Message, channel, "Something went wrong removing that counter;")
			return true
		}
		err = it.Delete()
		if err != nil {
			log.Error().
				Err(err).
				Str("subject", subject).
				Str("itemName", itemName).
				Msg("Error removing item")
			p.Bot.Send(c, bot.Message, channel, "Something went wrong removing that counter;")
			return true
		}

		p.Bot.Send(c, bot.Action, channel, fmt.Sprintf("chops a few %s out of his brain",
			itemName))
		return true

	} else if message.Command && parts[0] == "count" {
		var subject string
		var itemName string

		if len(parts) == 3 {
			// report count for parts[1]
			subject = strings.ToLower(parts[1])
			itemName = strings.ToLower(parts[2])
		} else if len(parts) == 2 {
			subject = strings.ToLower(nick)
			itemName = strings.ToLower(parts[1])
		} else {
			return false
		}

		var item Item
		item, err := GetItem(p.DB, subject, itemName)
		switch {
		case err == sql.ErrNoRows:
			p.Bot.Send(c, bot.Message, channel, fmt.Sprintf("I don't think %s has any %s.",
				subject, itemName))
			return true
		case err != nil:
			log.Error().
				Err(err).
				Str("subject", subject).
				Str("itemName", itemName).
				Msg("Error retrieving item count")
			return true
		}

		p.Bot.Send(c, bot.Message, channel, fmt.Sprintf("%s has %d %s.", subject, item.Count,
			itemName))

		return true
	} else if len(parts) <= 2 {
		if (len(parts) == 2) && (parts[1] == "++" || parts[1] == "--") {
			parts = []string{parts[0] + parts[1]}
		}
		// Need to have at least 3 characters to ++ or --
		if len(parts[0]) < 3 {
			return false
		}

		subject := strings.ToLower(nick)
		itemName := strings.ToLower(parts[0])[:len(parts[0])-2]

		if nameParts := strings.SplitN(itemName, ".", 2); len(nameParts) == 2 {
			subject = nameParts[0]
			itemName = nameParts[1]
		}

		if strings.HasSuffix(parts[0], "++") {
			// ++ those fuckers
			item, err := GetItem(p.DB, subject, itemName)
			if err != nil {
				log.Error().
					Err(err).
					Str("subject", subject).
					Str("itemName", itemName).
					Msg("error finding item")
				// Item ain't there, I guess
				return false
			}
			log.Debug().Msgf("About to update item: %#v", item)
			item.UpdateDelta(1)
			p.Bot.Send(c, bot.Message, channel, fmt.Sprintf("%s has %d %s.", subject,
				item.Count, item.Item))
			return true
		} else if strings.HasSuffix(parts[0], "--") {
			// -- those fuckers
			item, err := GetItem(p.DB, subject, itemName)
			if err != nil {
				log.Error().
					Err(err).
					Str("subject", subject).
					Str("itemName", itemName).
					Msg("Error finding item")
				// Item ain't there, I guess
				return false
			}
			item.UpdateDelta(-1)
			p.Bot.Send(c, bot.Message, channel, fmt.Sprintf("%s has %d %s.", subject,
				item.Count, item.Item))
			return true
		}
	} else if len(parts) == 3 {
		// Need to have at least 3 characters to ++ or --
		if len(parts[0]) < 3 {
			return false
		}

		subject := strings.ToLower(nick)
		itemName := strings.ToLower(parts[0])

		if nameParts := strings.SplitN(itemName, ".", 2); len(nameParts) == 2 {
			subject = nameParts[0]
			itemName = nameParts[1]
		}

		if parts[1] == "+=" {
			// += those fuckers
			item, err := GetItem(p.DB, subject, itemName)
			if err != nil {
				log.Error().
					Err(err).
					Str("subject", subject).
					Str("itemName", itemName).
					Msg("Error finding item")
				// Item ain't there, I guess
				return false
			}
			n, _ := strconv.Atoi(parts[2])
			log.Debug().Msgf("About to update item by %d: %#v", n, item)
			item.UpdateDelta(n)
			p.Bot.Send(c, bot.Message, channel, fmt.Sprintf("%s has %d %s.", subject,
				item.Count, item.Item))
			return true
		} else if parts[1] == "-=" {
			// -= those fuckers
			item, err := GetItem(p.DB, subject, itemName)
			if err != nil {
				log.Error().
					Err(err).
					Str("subject", subject).
					Str("itemName", itemName).
					Msg("Error finding item")
				// Item ain't there, I guess
				return false
			}
			n, _ := strconv.Atoi(parts[2])
			log.Debug().Msgf("About to update item by -%d: %#v", n, item)
			item.UpdateDelta(-n)
			p.Bot.Send(c, bot.Message, channel, fmt.Sprintf("%s has %d %s.", subject,
				item.Count, item.Item))
			return true
		}
	}

	return false
}

// Help responds to help requests. Every plugin must implement a help function.
func (p *CounterPlugin) help(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	p.Bot.Send(c, bot.Message, message.Channel, "You can set counters incrementally by using "+
		"<noun>++ and <noun>--. You can see all of your counters using "+
		"\"inspect\", erase them with \"clear\", and view single counters with "+
		"\"count\".")
	return true
}

func (p *CounterPlugin) checkMatch(c bot.Connector, message msg.Message) bool {
	nick := message.User.Name
	channel := message.Channel

	submatches := teaMatcher.FindStringSubmatch(message.Body)
	if len(submatches) <= 1 {
		return false
	}
	itemName := strings.ToLower(submatches[1])

	// We will specifically allow :tea: to keep compatability
	item, err := GetItem(p.DB, nick, itemName)
	if err != nil || (item.Count == 0 && item.Item != ":tea:") {
		log.Error().
			Err(err).
			Str("itemName", itemName).
			Msg("Error finding item")
		// Item ain't there, I guess
		return false
	}
	log.Debug().Msgf("About to update item: %#v", item)
	delta := 1
	if item.Count < 0 {
		delta = -1
	}
	item.UpdateDelta(delta)
	p.Bot.Send(c, bot.Message, channel, fmt.Sprintf("%s... %s has %d %s",
		strings.Join(everyDayImShuffling([]string{"bleep", "bloop", "blop"}), "-"), nick, item.Count, itemName))
	return true
}

func everyDayImShuffling(vals []string) []string {
	ret := make([]string, len(vals))
	perm := rand.Perm(len(vals))
	for i, randIndex := range perm {
		ret[i] = vals[randIndex]
	}
	return ret
}

func (p *CounterPlugin) registerWeb() {
	http.HandleFunc("/counter/api", p.handleCounterAPI)
	http.HandleFunc("/counter", p.handleCounter)
	p.Bot.RegisterWeb("/counter", "Counter")
}

var tpl = template.Must(template.New("factoidIndex").Parse(html))

func (p *CounterPlugin) handleCounter(w http.ResponseWriter, r *http.Request) {
	tpl.Execute(w, struct{ Nav []bot.EndPoint }{p.Bot.GetWebNavigation()})
}

func (p *CounterPlugin) handleCounterAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		info := struct {
			User     string
			Thing    string
			Action   string
			Password string
		}{}
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&info)
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprint(w, err)
			return
		}
		log.Debug().
			Interface("postbody", info).
			Msg("Got a POST")
		if info.Password != p.Bot.GetPassword() {
			w.WriteHeader(http.StatusForbidden)
			j, _ := json.Marshal(struct{ Err string }{Err: "Invalid Password"})
			w.Write(j)
			return
		}
		item, err := GetItem(p.DB, info.User, info.Thing)
		if err != nil {
			log.Error().
				Err(err).
				Str("subject", info.User).
				Str("itemName", info.Thing).
				Msg("error finding item")
			w.WriteHeader(404)
			fmt.Fprint(w, err)
			return
		}
		if info.Action == "++" {
			item.UpdateDelta(1)
		} else if info.Action == "--" {
			item.UpdateDelta(-1)
		} else {
			w.WriteHeader(400)
			fmt.Fprint(w, "Invalid increment")
			return
		}

	}
	all, err := GetAllItems(p.DB)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprint(w, err)
		return
	}
	data, err := json.Marshal(all)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprint(w, err)
		return
	}
	fmt.Fprint(w, string(data))
}
