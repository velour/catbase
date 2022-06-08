// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package cli

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/bot/user"
)

//go:embed *.html
var embeddedFS embed.FS

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
	r := chi.NewRouter()
	r.HandleFunc("/api", p.handleWebAPI)
	r.HandleFunc("/", p.handleWeb)
	p.bot.RegisterWebName(r, "/cli", "CLI")
}

func (p *CliPlugin) GetRouter() (http.Handler, string) {
	return nil, ""
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
	if !p.bot.CheckPassword("", info.Password) {
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

func (p *CliPlugin) handleWeb(w http.ResponseWriter, r *http.Request) {
	index, _ := embeddedFS.ReadFile("index.html")
	w.Write(index)
}

// Completing the Connector interface, but will not actually be a connector
func (p *CliPlugin) RegisterEvent(cb bot.Callback) {}
func (p *CliPlugin) Send(kind bot.Kind, args ...any) (string, error) {
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
func (p *CliPlugin) GetEmojiList(bool) map[string]string { return nil }
func (p *CliPlugin) Serve() error                        { return nil }
func (p *CliPlugin) Who(s string) []string               { return nil }
func (p *CliPlugin) Profile(name string) (user.User, error) {
	return user.User{}, fmt.Errorf("unimplemented")
}
func (p *CliPlugin) Emojy(name string) string             { return name }
func (p *CliPlugin) DeleteEmojy(name string) error        { return nil }
func (p *CliPlugin) UploadEmojy(emojy, path string) error { return nil }
func (p *CliPlugin) URLFormat(title, url string) string {
	return fmt.Sprintf("%s (%s)", title, url)
}
func (p *CliPlugin) GetChannelName(id string) string     { return id }
func (p *CliPlugin) GetChannelID(name string) string     { return name }
func (p *CliPlugin) GetRoles() ([]bot.Role, error)       { return []bot.Role{}, nil }
func (p *CliPlugin) SetRole(userID, roleID string) error { return nil }
