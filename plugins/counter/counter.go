package counter

import (
	"database/sql"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/config"

	"github.com/jmoiron/sqlx"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
)

// This is a counter plugin to count arbitrary things.

type CounterPlugin struct {
	b   bot.Bot
	db  *sqlx.DB
	cfg *config.Config
}

type Item struct {
	*sqlx.DB

	ID     int64
	Nick   string
	Item   string
	Count  int
	UserID string
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
	// Don't forget to embed the db into all of that shiz
	for i := range items {
		items[i].DB = db
	}
	return items, nil
}

// GetItems returns all counters for a subject
func GetItems(db *sqlx.DB, nick, id string) ([]Item, error) {
	var items []Item
	var err error
	if id != "" {
		err = db.Select(&items, `select * from counter where userid = ?`, id)
	} else {
		err = db.Select(&items, `select * from counter where nick = ?`, nick)
	}
	if err != nil {
		return nil, err
	}
	// Don't forget to embed the db into all of that shiz
	for i := range items {
		items[i].DB = db
	}
	return items, nil
}

func LeaderAll(db *sqlx.DB) ([]Item, error) {
	s := `select id,item,nick,count from (select id,item,nick,count,max(abs(count)) from counter group by item having count(nick) > 1 and max(abs(count)) > 1) order by count desc`
	var items []Item
	err := db.Select(&items, s)
	if err != nil {
		log.Error().Msgf("Error querying leaderboard: %s", err)
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

func RmAlias(db *sqlx.DB, aliasName string) error {
	q := `delete from counter_alias where item = ?`
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	_, err = tx.Exec(q, aliasName)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			return err
		}
		return err
	}
	err = tx.Commit()
	return err
}

func MkAlias(db *sqlx.DB, aliasName, pointsTo string) (*alias, error) {
	aliasName = strings.ToLower(aliasName)
	pointsTo = strings.ToLower(pointsTo)
	res, err := db.Exec(`insert into counter_alias (item, points_to) values (?, ?)`,
		aliasName, pointsTo)
	if err != nil {
		_, err := db.Exec(`update counter_alias set points_to=? where item=?`, pointsTo, aliasName)
		if err != nil {
			return nil, err
		}
		var a alias
		if err := db.Get(&a, `select * from counter_alias where item=?`, aliasName); err != nil {
			return nil, err
		}
		return &a, nil
	}
	id, _ := res.LastInsertId()

	return &alias{db, id, aliasName, pointsTo}, nil
}

// GetUserItem returns a specific counter for all subjects
func GetItem(db *sqlx.DB, itemName string) ([]Item, error) {
	itemName = trimUnicode(itemName)
	var items []Item
	var a alias
	if err := db.Get(&a, `select * from counter_alias where item=?`, itemName); err == nil {
		itemName = a.PointsTo
	} else {
		log.Error().Err(err).Interface("alias", a)
	}

	err := db.Select(&items, `select * from counter where item= ?`, itemName)
	if err != nil {
		return nil, err
	}
	log.Debug().
		Str("itemName", itemName).
		Interface("items", items).
		Msg("got item")
	for _, i := range items {
		i.DB = db
	}
	return items, nil
}

// GetUserItem returns a specific counter for a subject
func GetUserItem(db *sqlx.DB, nick, id, itemName string) (Item, error) {
	itemName = trimUnicode(itemName)
	var item Item
	item.DB = db
	var a alias
	if err := db.Get(&a, `select * from counter_alias where item=?`, itemName); err == nil {
		itemName = a.PointsTo
	} else {
		log.Error().Err(err).Interface("alias", a)
	}

	var err error
	if id != "" {
		err = db.Get(&item, `select * from counter where userid = ? and item= ?`, id, itemName)
	} else {
		err = db.Get(&item, `select * from counter where nick = ? and item= ?`, nick, itemName)
	}
	switch err {
	case sql.ErrNoRows:
		item.ID = -1
		item.Nick = nick
		item.Item = itemName
		item.UserID = id
	case nil:
	default:
		return Item{}, err
	}
	log.Debug().
		Str("nick", nick).
		Str("id", id).
		Str("itemName", itemName).
		Interface("item", item).
		Msg("got item")
	return item, nil
}

// GetUserItem returns a specific counter for a subject
// Create saves a counter
func (i *Item) Create() error {
	res, err := i.Exec(`insert into counter (nick, item, count, userid) values (?, ?, ?, ?);`,
		i.Nick, i.Item, i.Count, i.UserID)
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
func (i *Item) Update(r *bot.Request, value int) error {
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
	if err == nil {
		sendUpdate(r, i.Nick, i.Item, i.Count)
	}
	return err
}

// UpdateDelta changes a value according to some delta
// This will create or delete the item if necessary
func (i *Item) UpdateDelta(r *bot.Request, delta int) error {
	i.Count += delta
	return i.Update(r, i.Count)
}

// Delete removes a counter from the database
func (i *Item) Delete() error {
	_, err := i.Exec(`delete from counter where id = ?`, i.ID)
	i.ID = -1
	return err
}

func (p *CounterPlugin) migrate(r bot.Request) bool {
	db := p.db

	nicks := []string{}
	err := db.Select(&nicks, `select distinct nick from counter where userid is null`)
	if err != nil {
		log.Error().Err(err).Msg("could not get nick list")
		return false
	}

	log.Debug().Msgf("Migrating %d nicks to IDs", len(nicks))

	tx := db.MustBegin()

	for _, nick := range nicks {
		user, err := r.Conn.Profile(nick)
		if err != nil {
			continue
		}
		if _, err = tx.Exec(`update counter set userid=? where nick=?`, user.ID, nick); err != nil {
			log.Error().Err(err).Msg("Could not migrate users")
			continue
		}
	}

	if err := tx.Commit(); err != nil {
		log.Error().Err(err).Msg("Could not migrate users")
	}
	return false
}

func setupDB(b bot.Bot) error {
	db := b.DB()
	tx := db.MustBegin()
	db.MustExec(`create table if not exists counter (
			id integer primary key,
			nick string,
			item string,
			count integer
		);`)
	db.MustExec(`create table if not exists counter_alias (
			id integer PRIMARY KEY AUTOINCREMENT,
			item string NOT NULL UNIQUE,
			points_to string NOT NULL
		);`)
	tx.Commit()

	tx = db.MustBegin()
	count := 0
	err := tx.Get(&count, `SELECT count(*) FROM pragma_table_info('counter') where name='userid'`)
	if err != nil {
		return err
	}
	if count == 0 {
		tx.MustExec(`alter table counter add column userid string`)
	}
	tx.Commit()

	return nil
}

// NewCounterPlugin creates a new CounterPlugin with the Plugin interface
func New(b bot.Bot) *CounterPlugin {
	if err := setupDB(b); err != nil {
		panic(err)
	}

	cp := &CounterPlugin{
		b:  b,
		db: b.DB(),
	}

	b.RegisterRegex(cp, bot.Startup, regexp.MustCompile(`.*`), cp.migrate)

	b.RegisterRegexCmd(cp, bot.Message, mkAliasRegex, cp.mkAliasCmd)
	b.RegisterRegexCmd(cp, bot.Message, rmAliasRegex, cp.rmAliasCmd)
	b.RegisterRegexCmd(cp, bot.Message, leaderboardRegex, cp.leaderboardCmd)
	b.RegisterRegex(cp, bot.Message, teaRegex, cp.teaMatchCmd)
	b.RegisterRegexCmd(cp, bot.Message, resetRegex, cp.resetCmd)
	b.RegisterRegexCmd(cp, bot.Message, inspectRegex, cp.inspectCmd)
	b.RegisterRegexCmd(cp, bot.Message, clearRegex, cp.clearCmd)
	b.RegisterRegexCmd(cp, bot.Message, countRegex, cp.countCmd)
	b.RegisterRegex(cp, bot.Message, incrementRegex, cp.incrementCmd)
	b.RegisterRegex(cp, bot.Message, decrementRegex, cp.decrementCmd)
	b.RegisterRegex(cp, bot.Message, addToRegex, cp.addToCmd)
	b.RegisterRegex(cp, bot.Message, removeFromRegex, cp.removeFromCmd)

	b.Register(cp, bot.Help, cp.help)
	cp.registerWeb()

	return cp
}

func trimUnicode(s string) string {
	return strings.Trim(s, string(rune(0xFE0F)))
}

var mkAliasRegex = regexp.MustCompile(`(?i)^mkalias (?P<what>\S+) (?P<to>\S+)$`)
var rmAliasRegex = regexp.MustCompile(`(?i)^rmalias (?P<what>\S+)$`)
var leaderboardRegex = regexp.MustCompile(`(?i)^leaderboard\s?(?P<what>\S+)?$`)
var teaRegex = regexp.MustCompile("(?i)^([^.]+)\\. [^.]*\\. ([^.]*\\.?)+$")
var resetRegex = regexp.MustCompile(`(?i)^reset me$`)
var inspectRegex = regexp.MustCompile(`(?i)^inspect (?P<who>\S+)$`)
var clearRegex = regexp.MustCompile(`(?i)^clear (?P<what>\S+)$`)
var countRegex = regexp.MustCompile(`(?i)^count (?P<who>\S+)\s?(?P<what>\S+)?$`)
var incrementRegex = regexp.MustCompile(`(?i)^(?P<who>[^.\t\n\f\r ]+\s?\.)?(?P<thing>\S+)\s?\+\+$`)
var decrementRegex = regexp.MustCompile(`(?i)^(?P<who>[^.\t\n\f\r ]+\s?\.)?(?P<thing>\S+)\s?--$`)
var addToRegex = regexp.MustCompile(`(?i)^(?P<who>[^.\t\n\f\r ]+\s?\.)?(?P<thing>\S+)\s+\+=\s+(?P<amount>\d+)$`)
var removeFromRegex = regexp.MustCompile(`(?i)^(?P<who>[^.\t\n\f\r ]+\s?\.)?(?P<thing>\S+)\s+-=\s+(?P<amount>\d+)$`)

func (p *CounterPlugin) mkAliasCmd(r bot.Request) bool {
	what := r.Values["what"]
	to := r.Values["to"]
	if what == "" || to == "" {
		p.b.Send(r.Conn, bot.Message, fmt.Sprintf("You must provide all fields for an alias: %s", mkAliasRegex))
		return true
	}
	if _, err := MkAlias(p.db, what, to); err != nil {
		log.Error().Err(err).Msg("Could not mkalias")
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "We're gonna need too much db space to make an alias for your mom.")
		return true
	}
	p.b.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("Created alias %s -> %s",
		what, to))
	return true
}

func (p *CounterPlugin) rmAliasCmd(r bot.Request) bool {
	what := r.Values["what"]
	if what == "" {
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "You must specify an alias to remove.")
		return true
	}
	if err := RmAlias(p.db, what); err != nil {
		log.Error().Err(err).Msg("could not RmAlias")
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "`sudo rm your mom` => Nope, she's staying with me.")
		return true
	}
	p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "`sudo rm your mom`")
	return true
}

func (p *CounterPlugin) leaderboardCmd(r bot.Request) bool {
	var cmd func() ([]Item, error)
	itNameTxt := ""
	what := r.Values["what"]

	if what == "" {
		cmd = func() ([]Item, error) { return LeaderAll(p.db) }
	} else {
		itNameTxt = fmt.Sprintf(" for %s", what)
		cmd = func() ([]Item, error) { return Leader(p.db, what) }
	}

	its, err := cmd()
	if err != nil {
		log.Error().Err(err).Msg("Error with leaderboard")
		return false
	} else if len(its) == 0 {
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "There are not enough entries for a leaderboard.")
		return true
	}

	out := fmt.Sprintf("Leaderboard%s:\n", itNameTxt)
	for _, it := range its {
		out += fmt.Sprintf("%s with %d %s\n",
			it.Nick,
			it.Count,
			it.Item,
		)
	}
	p.b.Send(r.Conn, bot.Message, r.Msg.Channel, out)
	return true
}

func (p *CounterPlugin) resetCmd(r bot.Request) bool {
	nick, id := p.resolveUser(r, "")
	channel := r.Msg.Channel

	items, err := GetItems(p.db, nick, id)
	if err != nil {
		log.Error().
			Err(err).
			Str("nick", nick).
			Msg("Error getting items to reset")
		p.b.Send(r.Conn, bot.Message, channel, "Something is technically wrong with your counters.")
		return true
	}
	log.Debug().Msgf("Items: %+v", items)
	for _, item := range items {
		item.Delete()
	}
	p.b.Send(r.Conn, bot.Message, channel, fmt.Sprintf("%s, you are as new, my son.", nick))
	return true
}

func (p *CounterPlugin) inspectCmd(r bot.Request) bool {
	who := r.Values["who"]
	nick, id := "", ""
	if who == "me" {
		who = r.Msg.User.Name
		nick, id = p.resolveUser(r, "")
	} else {
		nick, id = p.resolveUser(r, who)
	}
	channel := r.Msg.Channel
	c := r.Conn

	log.Debug().
		Str("nick", nick).
		Str("id", id).
		Msg("Getting counter")
	// pull all of the items associated with "subject"
	items, err := GetItems(p.db, nick, id)
	if err != nil {
		log.Error().
			Err(err).
			Str("nick", nick).
			Str("id", id).
			Msg("Error retrieving items")
		p.b.Send(c, bot.Message, channel, "Something went wrong finding that counter;")
		return true
	}

	resp := fmt.Sprintf("%s has the following counters:", nick)
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
		p.b.Send(c, bot.Message, channel, fmt.Sprintf("%s has no counters.", nick))
		return true
	}

	p.b.Send(c, bot.Message, channel, resp)
	return true
}

func (p *CounterPlugin) clearCmd(r bot.Request) bool {
	nick, id := p.resolveUser(r, "")
	itemName := strings.ToLower(r.Values["what"])
	channel := r.Msg.Channel
	c := r.Conn

	it, err := GetUserItem(p.db, nick, id, itemName)
	if err != nil {
		log.Error().
			Err(err).
			Str("nick", nick).
			Str("id", id).
			Str("itemName", itemName).
			Msg("Error getting item to remove")
		p.b.Send(c, bot.Message, channel, "Something went wrong removing that counter;")
		return true
	}
	err = it.Delete()
	if err != nil {
		log.Error().
			Err(err).
			Str("nick", nick).
			Str("id", id).
			Str("itemName", itemName).
			Msg("Error removing item")
		p.b.Send(c, bot.Message, channel, "Something went wrong removing that counter;")
		return true
	}

	p.b.Send(c, bot.Action, channel, fmt.Sprintf("chops a few %s out of his brain",
		itemName))
	return true
}

func (p *CounterPlugin) countCmd(r bot.Request) bool {
	itemName := strings.ToLower(r.Values["what"])
	nick, id := r.Msg.User.Name, r.Msg.User.ID
	if r.Values["what"] == "" {
		itemName = r.Values["who"]
	} else {
		nick, id = p.resolveUser(r, r.Values["who"])
	}

	var item Item
	item, err := GetUserItem(p.db, nick, id, itemName)
	switch {
	case err == sql.ErrNoRows:
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("I don't think %s has any %s.",
			nick, itemName))
		return true
	case err != nil:
		log.Error().
			Err(err).
			Str("nick", nick).
			Str("id", id).
			Str("itemName", itemName).
			Msg("Error retrieving item count")
		return true
	}

	p.b.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("%s has %d %s.", nick, item.Count,
		itemName))

	return true
}

func (p *CounterPlugin) incrementCmd(r bot.Request) bool {
	nick, id := r.Msg.User.Name, r.Msg.User.ID
	if r.Values["who"] != "" {
		nick, id = p.resolveUser(r, r.Values["who"])
	}
	itemName := r.Values["thing"]
	channel := r.Msg.Channel
	// ++ those fuckers
	item, err := GetUserItem(p.db, nick, id, itemName)
	if err != nil {
		log.Error().
			Err(err).
			Str("nick", nick).
			Str("id", id).
			Str("itemName", itemName).
			Msg("error finding item")
		// Item ain't there, I guess
		return false
	}
	log.Debug().Msgf("About to update item: %#v", item)
	item.UpdateDelta(&r, 1)
	p.b.Send(r.Conn, bot.Message, channel, fmt.Sprintf("%s has %d %s.", nick,
		item.Count, item.Item))
	return true
}

func (p *CounterPlugin) decrementCmd(r bot.Request) bool {
	nick, id := r.Msg.User.Name, r.Msg.User.ID
	if r.Values["who"] != "" {
		nick, id = p.resolveUser(r, r.Values["who"])
	}
	itemName := r.Values["thing"]
	channel := r.Msg.Channel
	// -- those fuckers
	item, err := GetUserItem(p.db, nick, id, itemName)
	if err != nil {
		log.Error().
			Err(err).
			Str("nick", nick).
			Str("id", id).
			Str("itemName", itemName).
			Msg("Error finding item")
		// Item ain't there, I guess
		return false
	}
	item.UpdateDelta(&r, -1)
	p.b.Send(r.Conn, bot.Message, channel, fmt.Sprintf("%s has %d %s.", nick,
		item.Count, item.Item))
	return true
}

func (p *CounterPlugin) addToCmd(r bot.Request) bool {
	nick, id := p.resolveUser(r, r.Values["who"])
	itemName := r.Values["thing"]
	channel := r.Msg.Channel
	// += those fuckers
	item, err := GetUserItem(p.db, nick, id, itemName)
	if err != nil {
		log.Error().
			Err(err).
			Str("nick", nick).
			Str("id", id).
			Str("itemName", itemName).
			Msg("Error finding item")
		// Item ain't there, I guess
		return false
	}
	n, _ := strconv.Atoi(r.Values["amount"])
	log.Debug().Msgf("About to update item by %d: %#v", n, item)
	item.UpdateDelta(&r, n)
	p.b.Send(r.Conn, bot.Message, channel, fmt.Sprintf("%s has %d %s.", nick,
		item.Count, item.Item))
	return true
}

func (p *CounterPlugin) removeFromCmd(r bot.Request) bool {
	nick, id := p.resolveUser(r, r.Values["who"])
	itemName := r.Values["thing"]
	channel := r.Msg.Channel
	// -= those fuckers
	item, err := GetUserItem(p.db, nick, id, itemName)
	if err != nil {
		log.Error().
			Err(err).
			Str("nick", nick).
			Str("id", id).
			Str("itemName", itemName).
			Msg("Error finding item")
		// Item ain't there, I guess
		return false
	}
	n, _ := strconv.Atoi(r.Values["amount"])
	log.Debug().Msgf("About to update item by -%d: %#v", n, item)
	item.UpdateDelta(&r, -n)
	p.b.Send(r.Conn, bot.Message, channel, fmt.Sprintf("%s has %d %s.", nick,
		item.Count, item.Item))
	return true
}

// Help responds to help requests. Every plugin must implement a help function.
func (p *CounterPlugin) help(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	p.b.Send(c, bot.Message, message.Channel, "You can set counters incrementally by using "+
		"`<noun>++` and `<noun>--`. You can see all of your counters using "+
		"`inspect`, erase them with `clear`, and view single counters with "+
		"`count`.")
	p.b.Send(c, bot.Message, message.Channel, "You can create aliases with `!mkalias <alias> <original>`")
	p.b.Send(c, bot.Message, message.Channel, "You can remove aliases with `!rmalias <alias>`")
	return true
}

func (p *CounterPlugin) teaMatchCmd(r bot.Request) bool {
	nick, id := r.Msg.User.Name, r.Msg.User.ID
	channel := r.Msg.Channel

	submatches := teaRegex.FindStringSubmatch(r.Msg.Body)
	if len(submatches) <= 1 {
		return false
	}
	itemName := strings.ToLower(submatches[1])

	// We will specifically allow :tea: to keep compatability
	item, err := GetUserItem(p.db, nick, id, itemName)
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
	item.UpdateDelta(&r, delta)
	p.b.Send(r.Conn, bot.Message, channel, fmt.Sprintf("%s... %s has %d %s",
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

type updateFunc func(bot.Request, Update)

var updateFuncs = []updateFunc{}

func RegisterUpdate(f updateFunc) {
	log.Debug().Msgf("registering update func")
	updateFuncs = append(updateFuncs, f)
}

func sendUpdate(r *bot.Request, who, what string, amount int) {
	if r == nil {
		return
	}
	log.Debug().Msgf("sending updates to %d places", len(updateFuncs))
	for _, f := range updateFuncs {
		f(*r, Update{who, what, amount})
	}
}

func (p *CounterPlugin) resolveUser(r bot.Request, nick string) (string, string) {
	id := ""
	if nick != "" {
		nick = strings.TrimSuffix(nick, ".")
		u, err := r.Conn.Profile(nick)
		if err == nil && u.ID != "" {
			id = u.ID
		}
	} else if r.Msg.User != nil {
		nick, id = r.Msg.User.Name, r.Msg.User.ID
	}
	nick = strings.ToLower(nick)
	log.Debug().Msgf("resolveUser(%s, %s) => (%s, %s)", r.Msg.User.ID, nick, nick, id)
	return nick, id
}
