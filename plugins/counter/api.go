package counter

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
)

//go:embed *.html
var embeddedFS embed.FS

func (p *CounterPlugin) registerWeb() {
	r := chi.NewRouter()
	r.HandleFunc("/api/users/{user}/items/{item}/increment", p.mkIncrementAPI(1))
	r.HandleFunc("/api/users/{user}/items/{item}/decrement", p.mkIncrementAPI(-1))
	r.HandleFunc("/api", p.handleCounterAPI)
	r.HandleFunc("/", p.handleCounter)
	p.b.RegisterWebName(r, "/counter", "Counter")
}

func (p *CounterPlugin) mkIncrementAPI(delta int) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		userName := chi.URLParam(r, "user")
		itemName := chi.URLParam(r, "item")

		secret, pass, ok := r.BasicAuth()
		if !ok || !p.b.CheckPassword(secret, pass) {
			err := fmt.Errorf("unauthorized access")
			log.Error().
				Err(err).
				Msg("error authenticating user")
			w.WriteHeader(401)
			j, _ := json.Marshal(struct {
				Status bool
				Error  string
			}{false, err.Error()})
			fmt.Fprint(w, string(j))
			return
		}

		// Try to find an ID if possible
		u, err := p.b.DefaultConnector().Profile(userName)
		if err != nil {
			log.Error().Err(err).Msg("error finding user")
			w.WriteHeader(400)
			j, _ := json.Marshal(struct {
				Status bool
				Error  error
			}{false, err})
			fmt.Fprint(w, string(j))
			return
		}

		item, err := GetUserItem(p.db, userName, u.ID, itemName)
		if err != nil {
			log.Error().Err(err).Msg("error finding item")
			w.WriteHeader(400)
			j, _ := json.Marshal(struct {
				Status bool
				Error  error
			}{false, err})
			fmt.Fprint(w, string(j))
			return
		}
		req := &bot.Request{
			Conn: p.b.DefaultConnector(),
			Kind: bot.Message,
			Msg: msg.Message{
				User:        &u,
				ChannelName: "#API",
				Body:        fmt.Sprintf("%s += %d", itemName, delta),
				Time:        time.Now(),
			},
			Values: nil,
			Args:   nil,
		}
		item.UpdateDelta(req, delta)
		msg := fmt.Sprintf("%s changed their %s counter by %d for a total of %d via the amazing %s API",
			userName, itemName, delta, item.Count, p.cfg.Get("nick", "catbase"))
		for _, ch := range p.cfg.GetArray("channels", []string{}) {
			p.b.Send(p.b.DefaultConnector(), bot.Message, ch, msg)
			req.Msg.Channel = ch
			sendUpdate(req, userName, itemName, item.Count)
		}
		j, _ := json.Marshal(struct{ Status bool }{true})
		fmt.Fprint(w, string(j))
	}
}

func (p *CounterPlugin) handleCounter(w http.ResponseWriter, r *http.Request) {
	index, _ := embeddedFS.ReadFile("index.html")
	w.Write(index)
}

func (p *CounterPlugin) handleCounterAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		info := struct {
			User     string
			Thing    string
			Action   string
			Password string
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
		if p.b.CheckPassword("", info.Password) {
			w.WriteHeader(http.StatusForbidden)
			j, _ := json.Marshal(struct{ Err string }{Err: "Invalid Password"})
			w.Write(j)
			return
		}
		nick, id := p.resolveUser(bot.Request{Conn: p.b.DefaultConnector()}, info.User)
		item, err := GetUserItem(p.db, nick, id, info.Thing)
		if err != nil {
			log.Error().
				Err(err).
				Str("subject", info.User).
				Str("itemName", info.Thing).
				Msg("error finding item")
			w.WriteHeader(404)
			fmt.Fprint(w, err)
			return
		}
		if info.Action == "++" {
			item.UpdateDelta(nil, 1)
		} else if info.Action == "--" {
			item.UpdateDelta(nil, -1)
		} else {
			w.WriteHeader(400)
			fmt.Fprint(w, "Invalid increment")
			return
		}

	}
	all, err := GetAllItems(p.db)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprint(w, err)
		return
	}
	data, err := json.Marshal(all)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprint(w, err)
		return
	}
	fmt.Fprint(w, string(data))
}

// Update represents a change that gets sent off to other plugins such as goals
type Update struct {
	// Username to be displayed/recorded
	Who string `json:"who"`
	// Counter Item
	What string `json:"what"`
	// Total counter amount
	Amount int `json:"amount"`
}
