// © 2016 the CatBase Authors under the WTFPL license. See AUTHORS for the list of authors.

// Package inventory contains the plugin that allows the bot to pad messages
// See the bucket RFC inventory: http://wiki.xkcd.com/irc/Bucket_inventory
package inventory

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/jmoiron/sqlx"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/config"
)

type InventoryPlugin struct {
	*sqlx.DB
	bot      bot.Bot
	config   *config.Config
	handlers bot.HandlerTable
}

// New creates a new InventoryPlugin with the Plugin interface
func New(b bot.Bot) *InventoryPlugin {
	config := b.Config()

	p := &InventoryPlugin{
		DB:     b.DB(),
		bot:    b,
		config: config,
	}

	b.RegisterFilter("$item", p.itemFilter)
	b.RegisterFilter("$giveitem", p.giveItemFilter)

	p.DB.MustExec(`create table if not exists inventory (
			item string primary key
		);`)

	p.register()

	return p
}

func (p *InventoryPlugin) giveItemFilter(input string) string {
	for strings.Contains(input, "$giveitem") {
		item := p.random()
		input = strings.Replace(input, "$giveitem", item, 1)
		p.remove(item)
	}
	return input
}

func (p *InventoryPlugin) itemFilter(input string) string {
	for strings.Contains(input, "$item") {
		input = strings.Replace(input, "$item", p.random(), 1)
	}
	return input
}

func (p *InventoryPlugin) register() {
	nick := p.config.Get("nick", "bot")

	p.handlers = append(p.handlers, bot.HandlerSpec{Kind: bot.Message, IsCmd: true,
		Regex: regexp.MustCompile(`inventory`),
		Handler: func(r bot.Request) bool {
			items := p.getAll()
			say := "I'm not holding anything"
			if len(items) > 0 {
				log.Debug().Msgf("I think I have more than 0 items: %+v, len(items)=%d", items, len(items))
				say = fmt.Sprintf("I'm currently holding %s", strings.Join(items, ", "))
			}
			p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, say)
			return true
		},
	})

	h := func(r bot.Request) bool {
		log.Debug().Msgf("Adding item: %s", r.Values["item"])
		return p.addItem(r.Conn, r.Msg, r.Values["item"])
	}

	for _, r := range []*regexp.Regexp{
		regexp.MustCompile(`(?i)^take this (?P<item>.+)$`),
		regexp.MustCompile(`(?i)^have a (?P<item>.+)$`),
	} {
		p.handlers = append(p.handlers, bot.HandlerSpec{Kind: bot.Message, IsCmd: false,
			Regex:   r,
			Handler: h})
	}

	for _, r := range []*regexp.Regexp{
		regexp.MustCompile(fmt.Sprintf(`(?i)^puts (?P<item>\S+) in %s$`, nick)),
		regexp.MustCompile(fmt.Sprintf("(?i)^gives %s (?P<item>.+)$", nick)),
		regexp.MustCompile(fmt.Sprintf(`(?i)^gives (?P<item>\S+) to %s$`, nick)),
	} {
		p.handlers = append(p.handlers, bot.HandlerSpec{Kind: bot.Action, IsCmd: false,
			Regex:   r,
			Handler: h})
	}

	p.bot.RegisterTable(p, p.handlers)
}

func (p *InventoryPlugin) removeRandom() string {
	var name string
	err := p.QueryRow(`select item from inventory order by random() limit 1`).Scan(
		&name,
	)
	if err != nil {
		log.Error().Err(err).Msgf("Error finding random entry")
		return "IAMERROR"
	}
	_, err = p.Exec(`delete from inventory where item=?`, name)
	if err != nil {
		log.Error().Err(err).Msgf("Error finding random entry")
		return "IAMERROR"
	}
	return name
}

func (p *InventoryPlugin) count() int {
	var output int
	err := p.QueryRow(`select count(*) as count from inventory`).Scan(&output)
	if err != nil {
		log.Error().Err(err).Msg("Error checking for item")
		return -1
	}
	return output
}

func (p *InventoryPlugin) random() string {
	var name string
	err := p.QueryRow(`select item from inventory order by random() limit 1`).Scan(
		&name,
	)
	if err != nil {
		log.Error().Err(err).Msg("Error finding random entry")
		return "IAMERROR"
	}
	return name
}

func (p *InventoryPlugin) getAll() []string {
	rows, err := p.Queryx(`select item from inventory`)
	if err != nil {
		log.Error().Err(err).Msg("Error getting all items")
		return []string{}
	}
	output := []string{}
	for rows.Next() {
		var item string
		rows.Scan(&item)
		output = append(output, item)
	}
	rows.Close()
	return output
}

func (p *InventoryPlugin) exists(i string) bool {
	var output int
	err := p.QueryRow(`select count(*) as count from inventory where item=?`, i).Scan(&output)
	if err != nil {
		log.Error().Err(err).Msg("Error checking for item")
		return false
	}
	return output > 0
}

func (p *InventoryPlugin) remove(i string) {
	_, err := p.Exec(`delete from inventory where item=?`, i)
	if err != nil {
		log.Error().Msg("Error inserting new inventory item")
	}
}

func (p *InventoryPlugin) addItem(c bot.Connector, m msg.Message, i string) bool {
	if p.exists(i) {
		p.bot.Send(c, bot.Message, m.Channel, fmt.Sprintf("I already have %s.", i))
		return true
	}
	var removed string
	max := p.config.GetInt("inventory.max", 10)
	if p.count() > max {
		removed = p.removeRandom()
	}
	_, err := p.Exec(`INSERT INTO inventory (item) values (?)`, i)
	if err != nil {
		log.Error().Err(err).Msg("Error inserting new inventory item")
	}
	if removed != "" {
		p.bot.Send(c, bot.Action, m.Channel, fmt.Sprintf("dropped %s and took %s from %s", removed, i, m.User.Name))
	} else {
		p.bot.Send(c, bot.Action, m.Channel, fmt.Sprintf("takes %s from %s", i, m.User.Name))
	}
	return true
}
