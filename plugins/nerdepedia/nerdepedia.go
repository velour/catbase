// © 2019 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package nerdepedia

import (
	"bufio"
	"fmt"
	"html"
	"net/http"
	"regexp"
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

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

var (
	client HTTPClient
)

func init() {
	client = &http.Client{}
}

type NerdepediaPlugin struct {
	bot    bot.Bot
	config *config.Config
}

// NewNerdepediaPlugin creates a new NerdepediaPlugin with the Plugin interface
func New(b bot.Bot) *NerdepediaPlugin {
	np := &NerdepediaPlugin{
		bot:    b,
		config: b.Config(),
	}
	b.RegisterRegex(np, bot.Message, regexp.MustCompile(`.*`), np.message)
	b.Register(np, bot.Help, np.help)
	return np
}

func defaultSites() map[string]string {
	starWars := "http://starwars.wikia.com/wiki/Special:Random"
	starTrek := "http://memory-alpha.wikia.com/wiki/Special:Random"
	dune := "http://dune.wikia.com/wiki/Special:Random"
	lotr := "http://lotr.wikia.com/wiki/Special:Random"
	pokemon := "http://pokemon.wikia.com/wiki/Special:Random"

	return map[string]string{
		"may the force be with you": starWars,
		"help me obi-wan":           starWars,

		"beam me up scotty":     starTrek,
		"live long and prosper": starTrek,

		"bless the maker":     dune,
		"i must not fear":     dune,
		"the spice must flow": dune,

		"my precious":                          lotr,
		"one ring to rule them all":            lotr,
		"one does not simply walk into mordor": lotr,

		"pikachu i choose you": pokemon,
		"gotta catch em all":   pokemon,
	}
}

// Message responds to the bot hook on recieving messages.
// This function returns true if the plugin responds in a meaningful way to the users message.
// Otherwise, the function returns false and the bot continues execution of other plugins.
func (p *NerdepediaPlugin) message(r bot.Request) bool {
	c := r.Conn
	message := r.Msg
	lowerCase := strings.ToLower(message.Body)
	query := ""
	queries := p.config.GetMap("nerdepedia.sites", defaultSites())
	for k, v := range queries {
		if lowerCase == k {
			query = v
			break
		}
	}

	if query != "" {
		req, _ := http.NewRequest(http.MethodGet, query, nil)
		resp, err := client.Do(req)
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
				p.bot.Send(c, bot.Message, message.Channel, fmt.Sprintf("%s (%s)", description, link))
				return true
			}
		}
	}
	return false
}

// Help responds to help requests. Every plugin must implement a help function.
func (p *NerdepediaPlugin) help(c bot.Connector, kind bot.Kind, message msg.Message, args ...any) bool {
	p.bot.Send(c, bot.Message, message.Channel, "nerd stuff")
	return true
}
