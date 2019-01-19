// © 2019 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package nerdepedia

import (
	"bufio"
	"fmt"
	"html"
	"net/http"
	"strings"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/config"
)

const (
	descriptionPrefix = "<meta name=\"description\" content=\""
	linkPrefix        = "<link rel=\"canonical\" href=\""

	closingTagSuffix = "\" />"
)

type NerdepediaPlugin struct {
	bot    bot.Bot
	config *config.Config
}

// NewNerdepediaPlugin creates a new NerdepediaPlugin with the Plugin interface
func New(bot bot.Bot) *NerdepediaPlugin {
	return &NerdepediaPlugin{
		bot:    bot,
		config: bot.Config(),
	}
}

// Message responds to the bot hook on recieving messages.
// This function returns true if the plugin responds in a meaningful way to the users message.
// Otherwise, the function returns false and the bot continues execution of other plugins.
func (p *NerdepediaPlugin) Message(message msg.Message) bool {
	lowerCase := strings.ToLower(message.Body)
	query := ""
	if lowerCase == "may the force be with you" || lowerCase == "help me obi-wan" {
		query = "http://starwars.wikia.com/wiki/Special:Random"
	} else if lowerCase == "beam me up scotty" || lowerCase == "live long and prosper" {
		query = "http://memory-alpha.wikia.com/wiki/Special:Random"
	} else if lowerCase == "bless the maker" || lowerCase == "i must not fear" || lowerCase == "the spice must flow" {
		query = "http://dune.wikia.com/wiki/Special:Random"
	} else if lowerCase == "my precious" || lowerCase == "one ring to rule them all" || lowerCase == "one does not simply walk into mordor" {
		query = "http://lotr.wikia.com/wiki/Special:Random"
	} else if lowerCase == "pikachu i choose you" || lowerCase == "gotta catch em all" {
		query = "http://pokemon.wikia.com/wiki/Special:Random"
	}

	if query != "" {
		resp, err := http.Get(query)
		if err != nil {
			return false
		}
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		link := ""
		description := ""
		for scanner.Scan() {
			line := scanner.Text()
			if description == "" {
				index := strings.Index(line, descriptionPrefix)
				if index >= 0 {
					description = html.UnescapeString(html.UnescapeString(strings.TrimSuffix(strings.TrimPrefix(line, descriptionPrefix), closingTagSuffix)))
				}
			}
			if link == "" {
				index := strings.Index(line, linkPrefix)
				if index >= 0 {
					link = strings.TrimSuffix(strings.TrimPrefix(line, linkPrefix), closingTagSuffix)
				}
			}

			if description != "" && link != "" {
				p.bot.SendMessage(message.Channel, fmt.Sprintf("%s (%s)", description, link))
				return true
			}
		}
	}
	return false
}

// Help responds to help requests. Every plugin must implement a help function.
func (p *NerdepediaPlugin) Help(channel string, parts []string) {
	p.bot.SendMessage(channel, "nerd stuff")
}

// Empty event handler because this plugin does not do anything on event recv
func (p *NerdepediaPlugin) Event(kind string, message msg.Message) bool {
	return false
}

// Handler for bot's own messages
func (p *NerdepediaPlugin) BotMessage(message msg.Message) bool {
	return false
}

// Register any web URLs desired
func (p *NerdepediaPlugin) RegisterWeb() *string {
	return nil
}

func (p *NerdepediaPlugin) ReplyMessage(message msg.Message, identifier string) bool { return false }
