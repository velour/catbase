// © 2016 the CatBase Authors under the WTFPL license. See AUTHORS for the list of authors.

// Package inventory contains the plugin that allows the bot to pad messages
// See the bucket RFC inventory: http://wiki.xkcd.com/irc/Bucket_inventory
package inventory

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/config"
)

type InventoryPlugin struct {
	*sqlx.DB
	bot                bot.Bot
	config             *config.Config
	r1, r2, r3, r4, r5 *regexp.Regexp
}

// New creates a new InventoryPlugin with the Plugin interface
func New(bot bot.Bot) *InventoryPlugin {
	config := bot.Config()
	nick := config.Get("nick", "bot")
	r1, err := regexp.Compile("take this (.+)")
	checkerr(err)
	r2, err := regexp.Compile("have a (.+)")
	checkerr(err)
	r3, err := regexp.Compile(fmt.Sprintf("puts (.+) in %s([^a-zA-Z].*)?", nick))
	checkerr(err)
	r4, err := regexp.Compile(fmt.Sprintf("gives %s (.+)", nick))
	checkerr(err)
	r5, err := regexp.Compile(fmt.Sprintf("gives (.+) to %s([^a-zA-Z].*)?", nick))
	checkerr(err)

	p := InventoryPlugin{
		DB:     bot.DB(),
		bot:    bot,
		config: config,
		r1:     r1, r2: r2, r3: r3, r4: r4, r5: r5,
	}

	bot.RegisterFilter("$item", p.itemFilter)
	bot.RegisterFilter("$giveitem", p.giveItemFilter)

	_, err = p.DB.Exec(`create table if not exists inventory (
			item string primary key
		);`)

	if err != nil {
		log.Fatal(err)
	}

	return &p
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

func (p *InventoryPlugin) Message(message msg.Message) bool {
	m := message.Body
	log.Printf("inventory trying to read %+v", message)
	if message.Command {
		if strings.ToLower(m) == "inventory" {
			items := p.getAll()
			say := "I'm not holding anything"
			if len(items) > 0 {
				log.Printf("I think I have more than 0 items: %+v, len(items)=%d", items, len(items))
				say = fmt.Sprintf("I'm currently holding %s", strings.Join(items, ", "))
			}
			p.bot.SendMessage(message.Channel, say)
			return true
		}

		// <Randall> Bucket[:,] take this (.+)
		// <Randall> Bucket[:,] have a (.+)
		if matches := p.r1.FindStringSubmatch(m); len(matches) > 0 {
			log.Printf("Found item to add: %s", matches[1])
			return p.addItem(message, matches[1])
		}
		if matches := p.r2.FindStringSubmatch(m); len(matches) > 0 {
			log.Printf("Found item to add: %s", matches[1])
			return p.addItem(message, matches[1])
		}
	}
	if message.Action {
		log.Println("Inventory found an action")
		// * Randall puts (.+) in Bucket([^a-zA-Z].*)?
		// * Randall gives Bucket (.+)
		// * Randall gives (.+) to Bucket([^a-zA-Z].*)?

		if matches := p.r3.FindStringSubmatch(m); len(matches) > 0 {
			log.Printf("Found item to add: %s", matches[1])
			return p.addItem(message, matches[1])
		}
		if matches := p.r4.FindStringSubmatch(m); len(matches) > 0 {
			log.Printf("Found item to add: %s", matches[1])
			return p.addItem(message, matches[1])
		}
		if matches := p.r5.FindStringSubmatch(m); len(matches) > 0 {
			log.Printf("Found item to add: %s", matches[1])
			return p.addItem(message, matches[1])
		}
	}
	return false
}

func (p *InventoryPlugin) removeRandom() string {
	var name string
	err := p.QueryRow(`select item from inventory order by random() limit 1`).Scan(
		&name,
	)
	if err != nil {
		log.Printf("Error finding random entry: %s", err)
		return "IAMERROR"
	}
	_, err = p.Exec(`delete from inventory where item=?`, name)
	if err != nil {
		log.Printf("Error finding random entry: %s", err)
		return "IAMERROR"
	}
	return name
}

func (p *InventoryPlugin) count() int {
	var output int
	err := p.QueryRow(`select count(*) as count from inventory`).Scan(&output)
	if err != nil {
		log.Printf("Error checking for item: %s", err)
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
		log.Printf("Error finding random entry: %s", err)
		return "IAMERROR"
	}
	return name
}

func (p *InventoryPlugin) getAll() []string {
	rows, err := p.Queryx(`select item from inventory`)
	if err != nil {
		log.Printf("Error getting all items: %s", err)
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
		log.Printf("Error checking for item: %s", err)
		return false
	}
	return output > 0
}

func (p *InventoryPlugin) remove(i string) {
	_, err := p.Exec(`delete from inventory where item=?`, i)
	if err != nil {
		log.Printf("Error inserting new inventory item: %s", err)
	}
}

func (p *InventoryPlugin) addItem(m msg.Message, i string) bool {
	if p.exists(i) {
		p.bot.SendMessage(m.Channel, fmt.Sprintf("I already have %s.", i))
		return true
	}
	var removed string
	max := p.config.GetInt("inventory.max", 10)
	if p.count() > max {
		removed = p.removeRandom()
	}
	_, err := p.Exec(`INSERT INTO inventory (item) values (?)`, i)
	if err != nil {
		log.Printf("Error inserting new inventory item: %s", err)
	}
	if removed != "" {
		p.bot.SendAction(m.Channel, fmt.Sprintf("dropped %s and took %s from %s", removed, i, m.User.Name))
	} else {
		p.bot.SendAction(m.Channel, fmt.Sprintf("takes %s from %s", i, m.User.Name))
	}
	return true
}

func checkerr(e error) {
	if e != nil {
		log.Println(e)
	}
}

func (p *InventoryPlugin) Event(e string, message msg.Message) bool {
	return false
}

func (p *InventoryPlugin) BotMessage(message msg.Message) bool {
	return false
}

func (p *InventoryPlugin) Help(e string, m []string) {
}

func (p *InventoryPlugin) RegisterWeb() *string {
	// nothing to register
	return nil
}

func (p *InventoryPlugin) ReplyMessage(message msg.Message, identifier string) bool { return false }
