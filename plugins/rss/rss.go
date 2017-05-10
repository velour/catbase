package rss

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"

	"github.com/mmcdole/gofeed"
)

type RSSPlugin struct {
	Bot       bot.Bot
	cache     map[string]*cacheItem
	shelfLife time.Duration
}

type cacheItem struct {
	key        string
	data       string
	expiration time.Time
}

func New(bot bot.Bot) *RSSPlugin {
	return &RSSPlugin{
		Bot:       bot,
		cache:     map[string]*cacheItem{},
		shelfLife: time.Minute * 20,
	}
}

func (p *RSSPlugin) Message(message msg.Message) bool {
	tokens := strings.Fields(message.Body)
	numTokens := len(tokens)

	if numTokens == 2 && strings.ToLower(tokens[0]) == "rss" {
		if data, ok := p.cache[strings.ToLower(tokens[1])]; ok && time.Now().Before(data.expiration) {
			log.Printf("rss cache hit")
			p.Bot.SendMessage(message.Channel, data.data)
			return true
		} else {
			fp := gofeed.NewParser()
			feed, err := fp.ParseURL(tokens[1])
			if err != nil {
				p.Bot.SendMessage(message.Channel, fmt.Sprintf("RSS error: %s", err.Error()))
				return true
			}
			response := feed.Title
			for _, item := range feed.Items {
				response += fmt.Sprintf("\n%s", item.Title)
			}

			p.cache[strings.ToLower(tokens[1])] = &cacheItem{
				key:        strings.ToLower(tokens[1]),
				data:       response,
				expiration: time.Now().Add(p.shelfLife),
			}

			p.Bot.SendMessage(message.Channel, response)
			return true
		}
	}

	return false
}

func (p *RSSPlugin) LoadData() {
	// This bot has no data to load
}

// Help responds to help requests. Every plugin must implement a help function.
func (p *RSSPlugin) Help(channel string, parts []string) {
	p.Bot.SendMessage(channel, "try '!rss http://rss.cnn.com/rss/edition.rss'")
}

// Empty event handler because this plugin does not do anything on event recv
func (p *RSSPlugin) Event(kind string, message msg.Message) bool {
	return false
}

// Handler for bot's own messages
func (p *RSSPlugin) BotMessage(message msg.Message) bool {
	return false
}

// Register any web URLs desired
func (p *RSSPlugin) RegisterWeb() *string {
	return nil
}
