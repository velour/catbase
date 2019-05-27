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
	bot       bot.Bot
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

func New(b bot.Bot) *RSSPlugin {
	rss := &RSSPlugin{
		bot:       b,
		cache:     map[string]*cacheItem{},
		shelfLife: time.Minute * time.Duration(b.Config().GetInt("rss.shelfLife", 20)),
		maxLines:  b.Config().GetInt("rss.maxLines", 5),
	}
	b.Register(rss, bot.Message, rss.message)
	b.Register(rss, bot.Help, rss.help)
	return rss
}

func (p *RSSPlugin) message(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	tokens := strings.Fields(message.Body)
	numTokens := len(tokens)

	if numTokens == 2 && strings.ToLower(tokens[0]) == "rss" {
		if item, ok := p.cache[strings.ToLower(tokens[1])]; ok && time.Now().Before(item.expiration) {
			p.bot.Send(c, bot.Message, message.Channel, item.getCurrentPage(p.maxLines))
			return true
		} else {
			fp := gofeed.NewParser()
			feed, err := fp.ParseURL(tokens[1])
			if err != nil {
				p.bot.Send(c, bot.Message, message.Channel, fmt.Sprintf("RSS error: %s", err.Error()))
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

			p.bot.Send(c, bot.Message, message.Channel, item.getCurrentPage(p.maxLines))
			return true
		}
	}

	return false
}

// Help responds to help requests. Every plugin must implement a help function.
func (p *RSSPlugin) help(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	p.bot.Send(c, bot.Message, message.Channel, "try '!rss http://rss.cnn.com/rss/edition.rss'")
	return true
}
