// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package babbler

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/bot/user"
	"github.com/velour/catbase/config"
)

type BabblerPlugin struct {
	Bot      bot.Bot
	db       *sqlx.DB
	config   *config.Config
	babblers map[string]*babbler
}

type babbler struct {
	start  *node
	end    *node
	lookup map[string]*node
}

type node struct {
	wordFrequency int
	arcs          map[string]*arc
}

type arc struct {
	transitionFrequency int
	next                *node
}

func New(bot bot.Bot) *BabblerPlugin {
	plugin := &BabblerPlugin{
		Bot:      bot,
		db:       bot.DB(),
		config:   bot.Config(),
		babblers: map[string]*babbler{},
	}

	return plugin
}

func (p *BabblerPlugin) makeBabbler(newUser user.User) {
	name := newUser.Name
	babbler, err := getMarkovChain(p.db, name)
	if err == nil {
		p.babblers[name] = babbler
	} else {
		p.babblers[name] = newBabbler()
	}
}

func (p *BabblerPlugin) makeBabblers(newUser user.User) {
	users := p.Bot.Who(p.config.MainChannel)
	users = append(users, newUser)
	for _, name := range p.config.Babbler.DefaultUsers {
		users = append(users, user.New(name))
	}
	for _, u := range users {
		p.makeBabbler(u)
	}
}

func (p *BabblerPlugin) Message(message msg.Message) bool {
	if len(p.babblers) == 0 {
		p.makeBabblers(*message.User)
	} else if _, ok := p.babblers[message.User.Name]; !ok {
		p.makeBabbler(*message.User)
	}

	lowercase := strings.ToLower(message.Body)
	tokens := strings.Fields(lowercase)

	if len(tokens) == 2 && tokens[1] == "says" {
		saying := p.babble(tokens[0])
		if saying == "" {
			p.Bot.SendMessage(message.Channel, "Ze ain't said nothin'")
		}
		p.Bot.SendMessage(message.Channel, saying)
		return true
	} else if len(tokens) == 4 && strings.Index(lowercase, "initialize babbler for ") == 0 {
		who := tokens[3]
		if _, ok := p.babblers[who]; !ok {
			babbler, err := getMarkovChain(p.db, who)
			if err == nil {
				p.babblers[who] = babbler
			} else {
				p.babblers[who] = newBabbler()
			}
			p.Bot.SendMessage(message.Channel, "Okay.")
			return true
		}
	} else if strings.Index(lowercase, "batch learn for ") == 0 {
		who := tokens[3]
		if _, ok := p.babblers[who]; !ok {
			p.babblers[who] = newBabbler()
		}

		body := strings.Join(tokens[4:], " ")
		body = strings.ToLower(body)

		for _, a := range strings.Split(body, ".") {
			for _, b := range strings.Split(a, "!") {
				for _, c := range strings.Split(b, "?") {
					for _, d := range strings.Split(c, "\n") {
						trimmed := strings.TrimSpace(d)
						if trimmed != "" {
							addToMarkovChain(p.babblers[who], trimmed)
						}
					}
				}
			}
		}

		p.Bot.SendMessage(message.Channel, "Phew that was tiring.")
		return true
	} else {
		addToMarkovChain(p.babblers[message.User.Name], lowercase)
	}

	
	return false
}

func (p *BabblerPlugin) Help(channel string, parts []string) {
	p.Bot.SendMessage(channel, "initialize babbler for seabass\n\nseabass says")
}

func (p *BabblerPlugin) Event(kind string, message msg.Message) bool {
	return false
}

func (p *BabblerPlugin) BotMessage(message msg.Message) bool {
	return false
}

func (p *BabblerPlugin) RegisterWeb() *string {
	return nil
}

func addToMarkovChain(babble *babbler, phrase string) {
	words := strings.Fields(strings.ToLower(phrase))

	prev := babble.start
	prev.wordFrequency++
	for i := range words {
		// has this word been seen before
		if _, ok := babble.lookup[words[i]]; !ok {
			babble.lookup[words[i]] = &node{
				wordFrequency: 1,
				arcs:          map[string]*arc{},
			}
		} else {
			babble.lookup[words[i]].wordFrequency++
		}

		// has this word been seen after the previous word before
		if _, ok := prev.arcs[words[i]]; !ok {
			prev.arcs[words[i]] = &arc{
				transitionFrequency: 1,
				next:                babble.lookup[words[i]],
			}
		} else {
			prev.arcs[words[i]].transitionFrequency++
		}
		prev = babble.lookup[words[i]]
	}

	// has this word ended a fact before
	if _, ok := prev.arcs[""]; !ok {
		prev.arcs[""] = &arc{
			transitionFrequency: 1,
			next:                babble.end,
		}
	} else {
		prev.arcs[""].transitionFrequency++
	}
}

func newBabbler() *babbler {
	return &babbler{
		start: &node{
			wordFrequency: 0,
			arcs:          map[string]*arc{},
		},
		end: &node{
			wordFrequency: 0,
			arcs:          map[string]*arc{},
		},
		lookup: map[string]*node{},
	}
}

// this who string isn't escaped, just sooo, you know.
func getMarkovChain(db *sqlx.DB, who string) (*babbler, error) {
	query := fmt.Sprintf(`select tidbit from factoid where fact like '%s quotes';`, who)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}

	babble := newBabbler()

	for rows.Next() {

		var tidbit string
		err := rows.Scan(&tidbit)
		if err != nil {
			return nil, err
		}

		addToMarkovChain(babble, tidbit)
	}
	return babble, nil
}

func (p *BabblerPlugin) babble(who string) string {
	if babbler, ok := p.babblers[who]; ok {
		if len(babbler.start.arcs) == 0 {
			return ""
		}
		words := []string{}
		cur := babbler.start
		for cur != babbler.end {
			which := rand.Intn(cur.wordFrequency)
			sum := 0
			for word, arc := range cur.arcs {
				sum += arc.transitionFrequency
				if sum > which {
					words = append(words, word)
					cur = arc.next
					break
				}
			}
		}

		return strings.Join(words, " ")
	}

	return fmt.Sprintf("could not find babbler: %s", who)
}
