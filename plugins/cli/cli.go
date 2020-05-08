// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package cli

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/bot/user"
)

type CliPlugin struct {
	bot     bot.Bot
	db      *sqlx.DB
	cache   string
	counter int
}

func New(b bot.Bot) *CliPlugin {
	cp := &CliPlugin{
		bot: b,
	}
	cp.registerWeb()
	return cp
}

func (p *CliPlugin) registerWeb() {
	http.HandleFunc("/cli/api", p.handleWebAPI)
	http.HandleFunc("/cli", p.handleWeb)
	p.bot.RegisterWeb("/cli", "CLI")
}

func (p *CliPlugin) handleWebAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fmt.Fprintf(w, "Incorrect HTTP method")
		return
	}
	info := struct {
		User     string `json:"user"`
		Payload  string `json:"payload"`
		Password string `json:"password"`
	}{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&info)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprint(w, err)
		return
	}
	log.Debug().
		Interface("postbody", info).
		Msg("Got a POST")
	if info.Password != p.bot.GetPassword() {
		w.WriteHeader(http.StatusForbidden)
		j, _ := json.Marshal(struct{ Err string }{Err: "Invalid Password"})
		w.Write(j)
		return
	}

	p.bot.Receive(p, bot.Message, msg.Message{
		User: &user.User{
			ID:    info.User,
			Name:  info.User,
			Admin: false,
		},
		Channel: "web",
		Body:    info.Payload,
		Raw:     info.Payload,
		Command: true,
		Time:    time.Now(),
	})

	info.User = p.bot.WhoAmI()
	info.Payload = p.cache
	p.cache = ""

	data, err := json.Marshal(info)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprint(w, err)
		return
	}
	w.Write(data)
}

var tpl = template.Must(template.New("factoidIndex").Parse(indexHTML))

func (p *CliPlugin) handleWeb(w http.ResponseWriter, r *http.Request) {
	tpl.Execute(w, struct{ Nav []bot.EndPoint }{p.bot.GetWebNavigation()})
}

// Completing the Connector interface, but will not actually be a connector
func (p *CliPlugin) RegisterEvent(cb bot.Callback) {}
func (p *CliPlugin) Send(kind bot.Kind, args ...interface{}) (string, error) {
	switch kind {
	case bot.Message:
		fallthrough
	case bot.Action:
		fallthrough
	case bot.Reply:
		fallthrough
	case bot.Reaction:
		p.cache += args[1].(string) + "\n"
	}
	id := fmt.Sprintf("%d", p.counter)
	p.counter++
	return id, nil
}
func (p *CliPlugin) GetEmojiList() map[string]string { return nil }
func (p *CliPlugin) Serve() error                    { return nil }
func (p *CliPlugin) Who(s string) []string           { return nil }
func (s *CliPlugin) Profile(name string) (user.User, error) {
	return user.User{}, fmt.Errorf("unimplemented")
}
