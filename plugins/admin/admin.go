// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package admin

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
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
}

// Message responds to the bot hook on recieving messages.
// This function returns true if the plugin responds in a meaningful way to the users message.
// Otherwise, the function returns false and the bot continues execution of other plugins.
func (p *AdminPlugin) message(conn bot.Connector, k bot.Kind, message msg.Message, args ...interface{}) bool {
	body := message.Body

	if p.quiet {
		return true
	}

	if len(body) > 0 && body[0] == '$' {
		return p.handleVariables(conn, message)
	}

	if !message.Command {
		return false
	}

	if strings.ToLower(body) == "shut up" {
		dur := time.Duration(p.cfg.GetInt("quietDuration", 5)) * time.Minute
		log.Info().Msgf("Going to sleep for %v, %v", dur, time.Now().Add(dur))
		p.bot.Send(conn, bot.Message, message.Channel, "Okay. I'll be back later.")
		p.quiet = true
		go func() {
			select {
			case <-time.After(dur):
				p.quiet = false
				log.Info().Msg("Waking up from nap.")
			}
		}()
		return true
	}

	if strings.ToLower(body) == "password" {
		p.bot.Send(conn, bot.Message, message.Channel, p.bot.GetPassword())
		return true
	}

	parts := strings.Split(body, " ")
	if parts[0] == "set" && len(parts) > 2 && forbiddenKeys[parts[1]] {
		p.bot.Send(conn, bot.Message, message.Channel, "You cannot access that key")
		return true
	} else if parts[0] == "set" && len(parts) > 2 {
		p.cfg.Set(parts[1], strings.Join(parts[2:], " "))
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
