// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package admin

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/jmoiron/sqlx"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/config"
)

// This is a admin plugin to serve as an example and quick copy/paste for new plugins.

type AdminPlugin struct {
	bot bot.Bot
	db  *sqlx.DB
	cfg *config.Config

	quiet bool
}

// NewAdminPlugin creates a new AdminPlugin with the Plugin interface
func New(b bot.Bot) *AdminPlugin {
	p := &AdminPlugin{
		bot: b,
		db:  b.DB(),
		cfg: b.Config(),
	}
	b.Register(p, bot.Message, p.message)
	b.Register(p, bot.Help, p.help)
	p.registerWeb()
	return p
}

var forbiddenKeys = map[string]bool{
	"twitch.authorization": true,
	"twitch.clientid":      true,
	"untappd.token":        true,
	"slack.token":          true,
	"meme.memes":           true,
}

var addBlacklist = regexp.MustCompile(`(?i)disable plugin (.*)`)
var rmBlacklist = regexp.MustCompile(`(?i)enable plugin (.*)`)

// Message responds to the bot hook on recieving messages.
// This function returns true if the plugin responds in a meaningful way to the users message.
// Otherwise, the function returns false and the bot continues execution of other plugins.
func (p *AdminPlugin) message(conn bot.Connector, k bot.Kind, message msg.Message, args ...interface{}) bool {
	body := message.Body

	if p.quiet && message.Body != "come back" {
		return true
	}

	if len(body) > 0 && body[0] == '$' {
		return p.handleVariables(conn, message)
	}

	if !message.Command {
		return false
	}

	if p.quiet && message.Body == "come back" {
		p.quiet = false
		p.bot.SetQuiet(false)
		backMsg := p.bot.Config().Get("admin.comeback", "Okay, I'm back.")
		p.bot.Send(conn, bot.Message, message.Channel, backMsg)
		return true
	}

	if strings.ToLower(body) == "reboot" {
		p.bot.Send(conn, bot.Message, message.Channel, "brb")
		log.Info().Msgf("Got reboot command")
		os.Exit(0)
	}

	if strings.ToLower(body) == "shut up" {
		dur := time.Duration(p.cfg.GetInt("quietDuration", 5)) * time.Minute
		log.Info().Msgf("Going to sleep for %v, %v", dur, time.Now().Add(dur))
		leaveMsg := p.bot.Config().Get("admin.shutup", "Okay. I'll be back later.")
		p.bot.Send(conn, bot.Message, message.Channel, leaveMsg)
		p.quiet = true
		p.bot.SetQuiet(true)
		go func() {
			select {
			case <-time.After(dur):
				p.quiet = false
				p.bot.SetQuiet(false)
				log.Info().Msg("Waking up from nap.")
				backMsg := p.bot.Config().Get("admin.backmsg", "I'm back, bitches.")
				p.bot.Send(conn, bot.Message, message.Channel, backMsg)
			}
		}()
		return true
	}

	if addBlacklist.MatchString(body) {
		submatches := addBlacklist.FindStringSubmatch(message.Body)
		plugin := submatches[1]
		if err := p.addBlacklist(message.Channel, plugin); err != nil {
			p.bot.Send(conn, bot.Message, message.Channel, fmt.Sprintf("I couldn't add that item: %s", err))
			log.Error().Err(err).Msgf("error adding blacklist item")
			return true
		}
		p.bot.Send(conn, bot.Message, message.Channel, fmt.Sprintf("%s disabled. Use `!enable plugin %s` to re-enable it.", plugin, plugin))
		return true
	}

	if rmBlacklist.MatchString(body) {
		submatches := rmBlacklist.FindStringSubmatch(message.Body)
		plugin := submatches[1]
		if err := p.rmBlacklist(message.Channel, plugin); err != nil {
			p.bot.Send(conn, bot.Message, message.Channel, fmt.Sprintf("I couldn't remove that item: %s", err))
			log.Error().Err(err).Msgf("error removing blacklist item")
			return true
		}
		p.bot.Send(conn, bot.Message, message.Channel, fmt.Sprintf("%s enabled. Use `!disable plugin %s` to disable it.", plugin, plugin))
		return true
	}

	if strings.ToLower(body) == "password" {
		p.bot.Send(conn, bot.Message, message.Channel, p.bot.GetPassword())
		return true
	}

	verbs := map[string]bool{
		"set":    true,
		"push":   true,
		"setkey": true,
	}

	parts := strings.Split(body, " ")
	if verbs[parts[0]] && len(parts) > 2 && forbiddenKeys[parts[1]] {
		p.bot.Send(conn, bot.Message, message.Channel, "You cannot access that key")
		return true
	} else if parts[0] == "unset" && len(parts) > 1 {
		if err := p.cfg.Unset(parts[1]); err != nil {
			p.bot.Send(conn, bot.Message, message.Channel, fmt.Sprintf("Unset error: %s", err))
			return true
		}
		p.bot.Send(conn, bot.Message, message.Channel, fmt.Sprintf("Unset %s", parts[1]))
		return true
	} else if parts[0] == "set" && len(parts) > 2 {
		if err := p.cfg.Set(parts[1], strings.Join(parts[2:], " ")); err != nil {
			p.bot.Send(conn, bot.Message, message.Channel, fmt.Sprintf("Set error: %s", err))
			return true
		}
		p.bot.Send(conn, bot.Message, message.Channel, fmt.Sprintf("Set %s", parts[1]))
		return true
	} else if parts[0] == "push" && len(parts) > 2 {
		items := p.cfg.GetArray(parts[1], []string{})
		items = append(items, strings.Join(parts[2:], ""))
		if err := p.cfg.Set(parts[1], strings.Join(items, ";;")); err != nil {
			p.bot.Send(conn, bot.Message, message.Channel, fmt.Sprintf("Set error: %s", err))
			return true
		}
		p.bot.Send(conn, bot.Message, message.Channel, fmt.Sprintf("Set %s", parts[1]))
		return true
	} else if parts[0] == "setkey" && len(parts) > 3 {
		items := p.cfg.GetMap(parts[1], map[string]string{})
		items[parts[2]] = strings.Join(parts[3:], " ")
		if err := p.cfg.SetMap(parts[1], items); err != nil {
			p.bot.Send(conn, bot.Message, message.Channel, fmt.Sprintf("Set error: %s", err))
			return true
		}
		p.bot.Send(conn, bot.Message, message.Channel, fmt.Sprintf("Set %s", parts[1]))
		return true
	}

	if parts[0] == "get" && len(parts) == 2 && forbiddenKeys[parts[1]] {
		p.bot.Send(conn, bot.Message, message.Channel, "You cannot access that key")
		return true
	} else if parts[0] == "get" && len(parts) == 2 {
		v := p.cfg.Get(parts[1], "<unknown>")
		p.bot.Send(conn, bot.Message, message.Channel, fmt.Sprintf("%s: %s", parts[1], v))
		return true
	}

	return false
}

func (p *AdminPlugin) handleVariables(conn bot.Connector, message msg.Message) bool {
	if parts := strings.SplitN(message.Body, "!=", 2); len(parts) == 2 {
		variable := strings.ToLower(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])

		_, err := p.db.Exec(`delete from variables where name=? and value=?`, variable, value)
		if err != nil {
			p.bot.Send(conn, bot.Message, message.Channel, "I'm broke and need attention in my variable creation code.")
			log.Error().Err(err)
		} else {
			p.bot.Send(conn, bot.Message, message.Channel, "Removed.")
		}

		return true
	}

	parts := strings.SplitN(message.Body, "=", 2)
	if len(parts) != 2 {
		return false
	}

	variable := strings.ToLower(strings.TrimSpace(parts[0]))
	value := strings.TrimSpace(parts[1])

	var count int64
	row := p.db.QueryRow(`select count(*) from variables where value = ?`, variable, value)
	err := row.Scan(&count)
	if err != nil {
		p.bot.Send(conn, bot.Message, message.Channel, "I'm broke and need attention in my variable creation code.")
		log.Error().Err(err)
		return true
	}

	if count > 0 {
		p.bot.Send(conn, bot.Message, message.Channel, "I've already got that one.")
	} else {
		_, err := p.db.Exec(`INSERT INTO variables (name, value) VALUES (?, ?)`, variable, value)
		if err != nil {
			p.bot.Send(conn, bot.Message, message.Channel, "I'm broke and need attention in my variable creation code.")
			log.Error().Err(err)
			return true
		}
		p.bot.Send(conn, bot.Message, message.Channel, "Added.")
	}
	return true
}

// Help responds to help requests. Every plugin must implement a help function.
func (p *AdminPlugin) help(conn bot.Connector, kind bot.Kind, m msg.Message, args ...interface{}) bool {
	p.bot.Send(conn, bot.Message, m.Channel, "This does super secret things that you're not allowed to know about.")
	return true
}

func (p *AdminPlugin) registerWeb() {
	http.HandleFunc("/vars/api", p.handleWebAPI)
	http.HandleFunc("/vars", p.handleWeb)
	p.bot.RegisterWeb("/vars", "Variables")
}

var tpl = template.Must(template.New("factoidIndex").Parse(varIndex))

func (p *AdminPlugin) handleWeb(w http.ResponseWriter, r *http.Request) {
	tpl.Execute(w, struct{ Nav []bot.EndPoint }{p.bot.GetWebNavigation()})
}

func (p *AdminPlugin) handleWebAPI(w http.ResponseWriter, r *http.Request) {
	var configEntries []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	q := `select key, value from config`
	err := p.db.Select(&configEntries, q)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Error getting config entries.")
		w.WriteHeader(500)
		fmt.Fprint(w, err)
		return
	}
	for i, e := range configEntries {
		if strings.Contains(e.Value, ";;") {
			e.Value = strings.ReplaceAll(e.Value, ";;", ", ")
			e.Value = fmt.Sprintf("[%s]", e.Value)
			configEntries[i] = e
		}
	}
	j, _ := json.Marshal(configEntries)
	fmt.Fprintf(w, "%s", j)
}

func (p *AdminPlugin) addBlacklist(channel, plugin string) error {
	if plugin == "admin" {
		return fmt.Errorf("you cannot disable the admin plugin")
	}
	return p.modBlacklist(`insert or replace into pluginBlacklist values (?, ?)`, channel, plugin)
}

func (p *AdminPlugin) rmBlacklist(channel, plugin string) error {
	return p.modBlacklist(`delete from pluginBlacklist where channel=? and name=?`, channel, plugin)
}

func (p *AdminPlugin) modBlacklist(query, channel, plugin string) error {
	plugins := p.bot.GetPluginNames()
	for _, pp := range plugins {
		if pp == plugin {
			if _, err := p.db.Exec(query, channel, plugin); err != nil {
				return fmt.Errorf("%w", err)
			}
			err := p.bot.RefreshPluginBlacklist()
			if err != nil {
				return fmt.Errorf("%w", err)
			}
			return nil
		}
	}
	err := fmt.Errorf("unknown plugin named '%s'", plugin)
	return err
}
