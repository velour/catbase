package counter

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/bot/user"
)

func (p *CounterPlugin) registerWeb() {
	r := chi.NewRouter()
	r.HandleFunc("/api/users/{user}/items/{item}/increment", p.mkIncrementAPI(1))
	r.HandleFunc("/api/users/{user}/items/{item}/decrement", p.mkIncrementAPI(-1))
	r.HandleFunc("/api", p.handleCounterAPI)
	r.HandleFunc("/", p.handleCounter)
	p.b.RegisterWeb(r, "/counter", "Counter")
}

func (p *CounterPlugin) mkIncrementAPI(delta int) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := map[string]string{}
		userName := vars["user"]
		itemName := vars["item"]
		item, err := GetUserItem(p.db, userName, "", itemName)
		if err != nil {
			return
		}
		u := user.New(userName)
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
		msg := fmt.Sprintf("%s changed their %s counter by %d via the amazing %s API",
			userName, itemName, delta, p.cfg.Get("nick", "catbase"))
		for _, ch := range p.cfg.GetArray("channels", []string{}) {
			p.b.Send(p.b.DefaultConnector(), bot.Message, ch, msg)
		}
		j, _ := json.Marshal(struct{ Status bool }{true})
		fmt.Fprint(w, string(j))
	}
}

func (p *CounterPlugin) handleCounter(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, html)
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

type Update struct {
	Who    string
	What   string
	Amount int
}
