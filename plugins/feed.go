// © 2014 the AlePale Authors under the WTFPL license. See AUTHORS for the list of authors.

package plugins

import (
	"fmt"
	"log"
	"strings"
	"time"
)

import "github.com/chrissexton/alepale/bot"
import "labix.org/v2/mgo"
import "labix.org/v2/mgo/bson"
import "github.com/velour/feedme/webfeed"

// This is a skeleton plugin to serve as an example and quick copy/paste for new plugins.

type FeedPlugin struct {
	Bot  *bot.Bot
	Coll *mgo.Collection
}

// NewFeedPlugin creates a new FeedPlugin with the Plugin interface
func NewFeedPlugin(bot *bot.Bot) *FeedPlugin {
	// Mongo is removed, this plugin will crash if started
	log.Fatal("The Feed plugin has not been upgraded to SQL yet.")
	p := FeedPlugin{
		Bot: bot,
		// Coll: bot.Db.C("feed"),
	}
	go p.pollFeeds()
	return &p
}

// Message responds to the bot hook on recieving messages.
// This function returns true if the plugin responds in a meaningful way to the users message.
// Otherwise, the function returns false and the bot continues execution of other plugins.
func (p *FeedPlugin) Message(message bot.Message) bool {
	lowerBody := strings.ToLower(message.Body)
	parts := strings.Split(lowerBody, " ")

	log.Println(parts)
	if strings.HasPrefix(lowerBody, "register") && len(parts) >= 2 {
		url := parts[1]
		p.Coll.Insert(bson.M{"url": parts[1]})
		p.Bot.SendMessage(message.Channel, fmt.Sprintf("Watching %s", url))
		return true
	}
	return false
}

// Help responds to help requests. Every plugin must implement a help function.
func (p *FeedPlugin) Help(channel string, parts []string) {
	p.Bot.SendMessage(channel, "Sorry, Feed does not do a goddamn thing.")
}

// Empty event handler because this plugin does not do anything on event recv
func (p *FeedPlugin) Event(kind string, message bot.Message) bool {
	return false
}

// Handler for bot's own messages
func (p *FeedPlugin) BotMessage(message bot.Message) bool {
	return false
}

// Register any web URLs desired
func (p *FeedPlugin) RegisterWeb() *string {
	return nil
}

func (p *FeedPlugin) pollFeeds() {
	//duration := time.Duration(p.Bot.Config.QuoteTime) * time.Minute
	duration := time.Duration(20) * time.Second
	for {

		log.Println("checking feeds")

		// get all feeds
		var feeds []Feed
		err := p.Coll.Find(bson.M{"url": bson.M{"$exists": true}}).All(&feeds)
		if err != nil {
			log.Println("ERROR:", err)
		}
		log.Println("Feeds found:", feeds)
		for _, feed := range feeds {
			p.checkFeed(feed)
		}
		time.Sleep(time.Duration(duration))
	}

}

type Feed struct {
	Url  string
	feed webfeed.Feed
}

func (f Feed) Read(p []byte) (n int, err error) {
	var i int
	for i = 0; i < len(p) && i < len(f.Url); i++ {
		p[i] = f.Url[i]
	}

	return i, nil
}

func (p *FeedPlugin) checkFeed(f Feed) {
	log.Println("Checking feed:", f.Url)
	feed, err := webfeed.Read(f)
	if err != nil {
		log.Printf("Couldn't read %s: %s\n", f.Url, err)
		return
	}

	log.Println("Feed:", feed.Title)
	for _, entry := range feed.Entries {
		log.Println(entry)
	}
}
