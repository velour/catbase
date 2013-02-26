package plugins

import (
	"bitbucket.org/phlyingpenguin/godeepintir/bot"
	"bitbucket.org/phlyingpenguin/twitter"
	"github.com/garyburd/go-oauth/oauth"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
	"net/url"
	"strings"
	"time"
)

// Plugin to squak tweets at the channel.

type TwitterPlugin struct {
	Bot    *bot.Bot
	Coll   *mgo.Collection
	Client *twitter.Client
}

type twitterUser struct {
	Id     bson.ObjectId `bson:"_id,omitempty"`
	User   string
	Tweets []string
}

// NewTwitterPlugin creates a new TwitterPlugin with the Plugin interface
func NewTwitterPlugin(bot *bot.Bot) *TwitterPlugin {
	return &TwitterPlugin{
		Bot:  bot,
		Coll: bot.Db.C("twitter"),
	}
}

func (p *TwitterPlugin) say(message bot.Message, body string) bool {
	p.Bot.SendMessage(message.Channel, body)
	return true
}

// Check for commands accepted
// follow, unfollow
func (p *TwitterPlugin) checkCommand(message bot.Message) bool {
	parts := strings.Split(message.Body, " ")
	if parts[0] == "follow" {
		if len(parts) != 2 {
			return p.say(message, "I don't get it.")
		}

		user := parts[1]
		res := p.Coll.Find(bson.M{"user": user})

		if count, _ := res.Count(); count > 0 {
			return p.say(message, "I'm already following "+user+"!")
		}

		p.Coll.Insert(twitterUser{User: user})
	} else if parts[0] == "unfollow" {
		if len(parts) != 2 {
			return p.say(message, "I don't get it.")
		}

		user := parts[1]

		p.Coll.Remove(bson.M{"user": user})
		return p.say(message, "Fuck "+user)
	}

	return false
}

// Message responds to the bot hook on recieving messages.
// This function returns true if the plugin responds in a meaningful way to the users message.
// Otherwise, the function returns false and the bot continues execution of other plugins.
func (p *TwitterPlugin) Message(message bot.Message) bool {

	if message.Command {
		return p.checkCommand(message)
	}

	return false
}

// LoadData imports any configuration data into the plugin. This is not strictly necessary other
// than the fact that the Plugin interface demands it exist. This may be deprecated at a later
// date.
func (p *TwitterPlugin) LoadData() {
	// This bot has no data to load
	p.Client = twitter.New(&oauth.Credentials{
		p.Bot.Config.TwitterConsumerKey,
		p.Bot.Config.TwitterConsumerSecret,
	})

	p.Client.SetAuth(&oauth.Credentials{
		p.Bot.Config.TwitterUserKey,
		p.Bot.Config.TwitterUserSecret,
	})

	_, err := p.Client.VerifyCredentials(nil)

	if err != nil {
		log.Println("Could not auth with twitter:", err)
	} else {
		go p.checkMessages()
	}
}

// Help responds to help requests. Every plugin must implement a help function.
func (p *TwitterPlugin) Help(channel string, parts []string) {
	p.Bot.SendMessage(channel, "Tell me to follow or unfollow twitter users!")
}

// Empty event handler because this plugin does not do anything on event recv
func (p *TwitterPlugin) Event(kind string, message bot.Message) bool {
	return false
}

func (p *TwitterPlugin) checkMessages() {
	for {
		time.Sleep(time.Minute * 30)
		var u url.Values
		u.Set("screen_name", "phlyingpenguin")
		data, err := p.Client.UserTimeline(u)
		if err != nil {
			log.Println(err)
		} else {
			log.Println(data)
		}
	}
}
