// © 2013 the AlePale Authors under the WTFPL. See AUTHORS for the list of authors.

package plugins

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/chrissexton/alepale/bot"
)

// This is a skeleton plugin to serve as an example and quick copy/paste for new plugins.

type BeersPlugin struct {
	Bot *bot.Bot
	db  *sql.DB
}

type userBeers struct {
	id        int64
	nick      string
	count     int
	lastDrunk time.Time
	saved     bool
}

type untappdUser struct {
	id          int64
	untappdUser string
	channel     string
	lastCheckin int
	chanNick    string
}

// NewBeersPlugin creates a new BeersPlugin with the Plugin interface
func NewBeersPlugin(bot *bot.Bot) *BeersPlugin {
	if bot.DBVersion == 1 {
		if _, err := bot.DB.Exec(`create table if not exists beers (
			id integer primary key,
			nick string,
			count integer,
			lastDrunk integer
		);`); err != nil {
			log.Fatal(err)
		}

		if _, err := bot.DB.Exec(`create table if not exists untappd (
			id integer primary key,
			untappdUser string
			channel string
			lastCheckin integer
			chanNick string
		);`); err != nil {
			log.Fatal(err)
		}
	}
	p := BeersPlugin{
		Bot: bot,
		db:  bot.DB,
	}
	p.LoadData()
	for _, channel := range bot.Config.UntappdChannels {
		go p.untappdLoop(channel)
	}
	return &p
}

func (u *userBeers) Save(db *sql.DB) error {
	if !u.saved {
		res, err := db.Exec(`insert into beers (
			nick string,
			count integer,
			lastDrunk integer
		) values (?, ?, ?)`, u.nick, u.count, u.lastDrunk)
		if err != nil {
			return err
		}
		id, err := res.LastInsertId()
		if err != nil {
			return err
		}
		u.id = id
	}
	return nil
}

func getUserBeers(db *sql.DB, nick string) *userBeers {
	var ub userBeers
	err := db.QueryRow(`select id, nick, count, lastDrunk from beers
		where nick = ?`, nick).Scan(
		&ub.id,
		&ub.nick,
		&ub.count,
		&ub.lastDrunk,
	)
	if err != nil && err != sql.ErrNoRows {
		log.Println(err)
		return nil
	}

	return &ub
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
	parts[0] = strings.ToLower(parts[0]) // support iPhone/Android saying "Beers"
	if parts[0] == "beers" {
		if len(parts) == 3 {

			// try to get a count out of parts[2]
			count, err := strconv.Atoi(parts[2])
			if err != nil {
				// if it's not a number, maybe it's a nick!
				p.Bot.SendMessage(channel, "Sorry, that didn't make any sense.")
			}

			if count < 0 {
				// you can't be negative
				msg := fmt.Sprintf("Sorry %s, you can't have negative beers!", nick)
				p.Bot.SendMessage(channel, msg)
			}
			if parts[1] == "+=" {
				p.setBeers(nick, p.getBeers(nick)+count)
				p.randomReply(channel)
			} else if parts[1] == "=" {
				if count == 0 {
					p.puke(nick, channel)
				} else {
					p.setBeers(nick, count)
					p.randomReply(channel)
				}
			} else {
				p.Bot.SendMessage(channel, "I don't know your math.")
			}
		} else if len(parts) == 2 {
			if p.doIKnow(parts[1]) {
				p.reportCount(parts[1], channel, false)
			} else {
				msg := fmt.Sprintf("Sorry, I don't know %s.", parts[1])
				p.Bot.SendMessage(channel, msg)
			}
		} else if len(parts) == 1 {
			p.reportCount(nick, channel, true)
		}

		// no matter what, if we're in here, then we've responded
		return true
	} else if parts[0] == "beers--" {
		p.setBeers(nick, p.getBeers(nick)-1)
		p.Bot.SendAction(channel, "flushes")
		return true
	} else if parts[0] == "beers++" {
		p.addBeers(nick)
		p.randomReply(channel)
		return true
	} else if parts[0] == "bourbon++" {
		p.addBeers(nick)
		p.addBeers(nick)
		p.randomReply(channel)
		return true
	} else if parts[0] == "puke" {
		p.puke(nick, channel)
		return true
	}

	if message.Command && parts[0] == "imbibe" {
		p.addBeers(nick)
		p.randomReply(channel)
		return true
	}

	if message.Command && parts[0] == "reguntappd" {
		chanNick := message.User.Name
		channel := message.Channel

		if len(parts) < 2 {
			p.Bot.SendMessage(channel, "You must also provide a user name.")
		} else if len(parts) == 3 {
			chanNick = parts[2]
		} else if len(parts) == 4 {
			chanNick = parts[2]
			channel = parts[3]
		}
		u := untappdUser{
			untappdUser: parts[1],
			chanNick:    chanNick,
			channel:     channel,
		}

		log.Println("Creating Untappd user:", u.untappdUser, "nick:", u.chanNick)

		var count int
		err := p.db.QueryRow(`select count(*) from untappd 
			where untappdUser = ?`, u.untappdUser).Scan(&count)
		if err != nil {
			log.Println("Error registering untappd: ", err)
		}
		if count > 0 {
			p.Bot.SendMessage(channel, "I'm already watching you.")
			return true
		}
		_, err = p.db.Exec(`insert into untappd (
			untappdUser,
			channel,
			lastCheckin,
			chanNick
		) values (?, ?, ?, ?);`)
		if err != nil {
			log.Println("Error registering untappd: ", err)
			p.Bot.SendMessage(channel, "I can't see.")
			return true
		}

		p.Bot.SendMessage(channel, "I'll be watching you.")

		p.checkUntappd(channel)

		return true
	}

	if message.Command && parts[0] == "checkuntappd" {
		p.checkUntappd(channel)
		return true
	}

	return false
}

// Empty event handler because this plugin does not do anything on event recv
func (p *BeersPlugin) Event(kind string, message bot.Message) bool {
	return false
}

// LoadData imports any configuration data into the plugin. This is not strictly necessary other
// than the fact that the Plugin interface demands it exist. This may be deprecated at a later
// date.
func (p *BeersPlugin) LoadData() {
	rand.Seed(time.Now().Unix())
}

// Help responds to help requests. Every plugin must implement a help function.
func (p *BeersPlugin) Help(channel string, parts []string) {
	msg := "Beers: imbibe by using either beers +=,=,++ or with the !imbibe/drink " +
		"commands. I'll keep a count of how many beers you've had and then if you want " +
		"to reset, just !puke it all up!"
	p.Bot.SendMessage(channel, msg)
}

func (p *BeersPlugin) setBeers(user string, amount int) {
	ub := getUserBeers(p.db, user)
	ub.count = amount
	ub.lastDrunk = time.Now()
	ub.Save(p.db)
}

func (p *BeersPlugin) addBeers(user string) {
	p.setBeers(user, p.getBeers(user)+1)
}

func (p *BeersPlugin) getBeers(nick string) int {
	ub := getUserBeers(p.db, nick)
	return ub.count
}

func (p *BeersPlugin) reportCount(nick, channel string, himself bool) {
	beers := p.getBeers(nick)
	msg := fmt.Sprintf("%s has had %d beers so far.", nick, beers)
	if himself {
		if beers == 0 {
			msg = fmt.Sprintf("You really need to get drinkin, %s!", nick)
		} else {
			msg = fmt.Sprintf("You've had %d beers so far, %s.", beers, nick)
		}
	}
	p.Bot.SendMessage(channel, msg)
}

func (p *BeersPlugin) puke(user string, channel string) {
	p.setBeers(user, 0)
	msg := fmt.Sprintf("Ohhhhhh, and a reversal of fortune for %s!", user)
	p.Bot.SendMessage(channel, msg)
}

func (p *BeersPlugin) doIKnow(nick string) bool {
	var count int
	err := p.db.QueryRow(`select count(*) from beers where nick = ?`, nick).Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
}

// Sends random affirmation to the channel. This could be better (with a datastore for sayings)
func (p *BeersPlugin) randomReply(channel string) {
	replies := []string{"ZIGGY! ZAGGY!", "HIC!", "Stay thirsty, my friend!"}
	p.Bot.SendMessage(channel, replies[rand.Intn(len(replies))])
}

type checkin struct {
	Checkin_id      int
	Created_at      string
	Checkin_comment string
	// Rating_score    int // Apparently Untappd makes this a string when not present. :(
	Beer    map[string]interface{}
	Brewery map[string]interface{}
	Venue   interface{}
	User    mrUntappd
}

type mrUntappd struct {
	Uid          int
	User_name    string
	Relationship string
}

type checkins struct {
	Count int
	Items []checkin
}

type resp struct {
	Checkins checkins
}

type Beers struct {
	Response resp
}

func (p *BeersPlugin) pullUntappd() ([]checkin, error) {
	access_token := "?access_token=" + p.Bot.Config.UntappdToken
	baseUrl := "https://api.untappd.com/v4/checkin/recent/"

	url := baseUrl + access_token + "&limit=25"

	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode == 500 {
		return []checkin{}, errors.New(resp.Status)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	var beers Beers
	err = json.Unmarshal(body, &beers)
	if err != nil {
		log.Println(err)
		return []checkin{}, errors.New("Unable to parse JSON")
	}
	return beers.Response.Checkins.Items, nil
}

func (p *BeersPlugin) checkUntappd(channel string) {
	if p.Bot.Config.UntappdToken == "<Your Token>" {
		log.Println("No Untappd token, cannot enable plugin.")
		return
	}

	var users []untappdUser
	rows, err := p.db.Query(`select *from untappd`)
	if err != nil {
		log.Println("Error getting untappd users: ", err)
		return
	}
	for rows.Next() {
		u := untappdUser{}
		rows.Scan(&u.id, &u.untappdUser, &u.channel, &u.lastCheckin, &u.chanNick)
		users = append(users, u)
	}

	userMap := make(map[string]untappdUser)
	for _, u := range users {
		userMap[u.untappdUser] = u
	}

	chks, err := p.pullUntappd()
	for i := len(chks); i > 0; i-- {
		checkin := chks[i-1]

		if err != nil {
			log.Println("ERROR!:", err)
			continue
		}

		if checkin.Checkin_id <= userMap[checkin.User.User_name].lastCheckin {
			continue
		}

		venue := ""
		switch v := checkin.Venue.(type) {
		case map[string]interface{}:
			venue = " at " + v["venue_name"].(string)
		}
		beerName := checkin.Beer["beer_name"].(string)
		breweryName := checkin.Brewery["brewery_name"].(string)
		user, ok := userMap[checkin.User.User_name]
		if !ok {
			continue
		}
		p.addBeers(user.chanNick)
		drunken := p.getBeers(user.chanNick)

		msg := fmt.Sprintf("%s just drank %s by %s%s, bringing his drunkeness to %d",
			user.chanNick, beerName, breweryName, venue, drunken)
		if checkin.Checkin_comment != "" {
			msg = fmt.Sprintf("%s -- %s",
				msg, checkin.Checkin_comment)
		}

		user.lastCheckin = checkin.Checkin_id
		_, err := p.db.Exec(`update untappd set
			lastCheckin = ?
		where id = ?`, user.lastCheckin, user.id)
		if err != nil {
			log.Println("UPDATE ERROR!:", err)
		}

		log.Println("checkin id:", checkin.Checkin_id, "Message:", msg)
		p.Bot.SendMessage(channel, msg)
	}
}

func (p *BeersPlugin) untappdLoop(channel string) {
	frequency := p.Bot.Config.UntappdFreq

	log.Println("Checking every ", frequency, " seconds")

	for {
		time.Sleep(time.Duration(frequency) * time.Second)
		p.checkUntappd(channel)
	}
}

// Handler for bot's own messages
func (p *BeersPlugin) BotMessage(message bot.Message) bool {
	return false
}

// Register any web URLs desired
func (p *BeersPlugin) RegisterWeb() *string {
	return nil
}
