// © 2013 the AlePale Authors under the WTFPL. See AUTHORS for the list of authors.

package plugins

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/chrissexton/alepale/bot"
	"io/ioutil"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// This is a skeleton plugin to serve as an example and quick copy/paste for new plugins.

type BeersPlugin struct {
	Bot  *bot.Bot
	Coll *mgo.Collection
}

// NewBeersPlugin creates a new BeersPlugin with the Plugin interface
func NewBeersPlugin(bot *bot.Bot) *BeersPlugin {
	p := BeersPlugin{
		Bot: bot,
	}
	p.LoadData()
	for _, channel := range bot.Config.UntappdChannels {
		go p.untappdLoop(channel)
	}
	return &p
}

type userBeers struct {
	Nick      string
	Beercount int
	Lastdrunk time.Time
	Momentum  float64
	New       bool
}

func (u *userBeers) Save(coll *mgo.Collection) {
	_, err := coll.Upsert(bson.M{"nick": u.Nick}, u)
	if err != nil {
		panic(err)
	}
}

func getUserBeers(coll *mgo.Collection, nick string) *userBeers {
	ub := userBeers{New: true}
	coll.Find(bson.M{"nick": nick}).One(&ub)
	if ub.New == true {
		ub.New = false
		ub.Nick = nick
		ub.Beercount = 0
		ub.Momentum = 0
		ub.Save(coll)
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
			UntappdUser: parts[1],
			ChanNick:    chanNick,
			Channel:     channel,
		}

		log.Println("Creating Untappd user:", u.UntappdUser, "nick:", u.ChanNick)

		_, err := p.Coll.Upsert(bson.M{"untappduser": u.UntappdUser}, u)
		if err != nil {
			log.Println("ERROR!!!:", err)
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
	p.Coll = p.Bot.Db.C("beers")
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
	ub := getUserBeers(p.Coll, user)
	ub.Beercount = amount
	ub.Lastdrunk = time.Now()
	ub.Save(p.Coll)
}

func (p *BeersPlugin) addBeers(user string) {
	p.setBeers(user, p.getBeers(user)+1)
}

func (p *BeersPlugin) getBeers(nick string) int {
	ub := getUserBeers(p.Coll, nick)

	return ub.Beercount
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
	count, err := p.Coll.Find(bson.M{"nick": nick}).Count()
	if err != nil {
		panic(err)
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

type untappdUser struct {
	Id            bson.ObjectId `bson:"_id,omitempty"`
	UntappdUser   string
	Channel       string
	LastCheckin   int
	ChanNick      string
	KnownCheckins [5]int
}

func (p *BeersPlugin) pullUntappd() ([]checkin, error) {
	access_token := "?access_token=" + p.Bot.Config.UntappdToken
	baseUrl := "http://api.untappd.com/v4/checkin/recent/"

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
	p.Coll.Find(bson.M{"untappduser": bson.M{"$exists": true}, "channel": channel}).All(&users)

	userMap := make(map[string]untappdUser)
	for _, u := range users {
		userMap[u.UntappdUser] = u
	}

	chks, err := p.pullUntappd()
	for i := len(chks); i > 0; i-- {
		checkin := chks[i-1]

		if err != nil {
			log.Println("ERROR!:", err)
			continue
		}

		if checkin.Checkin_id <= userMap[checkin.User.User_name].LastCheckin {
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
		p.addBeers(user.ChanNick)
		drunken := p.getBeers(user.ChanNick)

		msg := fmt.Sprintf("%s just drank %s by %s%s, bringing his drunkeness to %d",
			user.ChanNick, beerName, breweryName, venue, drunken)
		if checkin.Checkin_comment != "" {
			msg = fmt.Sprintf("%s -- %s",
				msg, checkin.Checkin_comment)
		}

		user.LastCheckin = checkin.Checkin_id
		err := p.Coll.Update(bson.M{"_id": user.Id}, user)
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
