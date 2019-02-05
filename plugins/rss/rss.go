package rss

import (
	"fmt"
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
	maxLines  int
}

type cacheItem struct {
	key         string
	data        []string
	currentLine int
	expiration  time.Time
}

func (c *cacheItem) getCurrentPage(maxLines int) string {
	if len(c.data) <= maxLines {
		return strings.Join(c.data, "\n")
	}

	start := c.currentLine
	end := start + maxLines
	if end > len(c.data) {
		end = len(c.data)
	}

	page := strings.Join(c.data[start:end], "\n")

	if end-start == maxLines {
		c.currentLine = end
	} else {
		c.currentLine = maxLines - (end - start)
		page += "\n"
		page += strings.Join(c.data[0:c.currentLine], "\n")
	}

	return page
}

func New(bot bot.Bot) *RSSPlugin {
	return &RSSPlugin{
		Bot:       bot,
		cache:     map[string]*cacheItem{},
		shelfLife: time.Minute * 20,
		maxLines:  5,
	}
}

func (p *RSSPlugin) Message(message msg.Message) bool {
	tokens := strings.Fields(message.Body)
	numTokens := len(tokens)

	if numTokens == 2 && strings.ToLower(tokens[0]) == "rss" {
		if item, ok := p.cache[strings.ToLower(tokens[1])]; ok && time.Now().Before(item.expiration) {
			p.Bot.Send(bot.Message, message.Channel, item.getCurrentPage(p.maxLines))
			return true
		} else {
			fp := gofeed.NewParser()
			feed, err := fp.ParseURL(tokens[1])
			if err != nil {
				p.Bot.Send(bot.Message, message.Channel, fmt.Sprintf("RSS error: %s", err.Error()))
				return true
			}
			item := &cacheItem{
				key:         strings.ToLower(tokens[1]),
				data:        []string{feed.Title},
				expiration:  time.Now().Add(p.shelfLife),
				currentLine: 0,
			}

			for _, feedItem := range feed.Items {
				item.data = append(item.data, feedItem.Title)
			}

			p.cache[strings.ToLower(tokens[1])] = item

			p.Bot.Send(bot.Message, message.Channel, item.getCurrentPage(p.maxLines))
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
	p.Bot.Send(bot.Message, channel, "try '!rss http://rss.cnn.com/rss/edition.rss'")
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

func (p *RSSPlugin) ReplyMessage(message msg.Message, identifier string) bool { return false }
