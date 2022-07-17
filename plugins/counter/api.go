package counter

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
)

//go:embed *.html
var embeddedFS embed.FS

func (p *CounterPlugin) registerWeb() {
	r := chi.NewRouter()
	requests := p.cfg.GetInt("counter.requestsPer", 1)
	seconds := p.cfg.GetInt("counter.seconds", 1)
	r.Use(httprate.LimitByIP(requests, time.Duration(seconds)*time.Second))
	r.HandleFunc("/api/users/{user}/items/{item}/increment/{delta}", p.mkIncrementByNAPI(1))
	r.HandleFunc("/api/users/{user}/items/{item}/decrement/{delta}", p.mkIncrementByNAPI(-1))
	r.HandleFunc("/api/users/{user}/items/{item}/increment", p.mkIncrementAPI(1))
	r.HandleFunc("/api/users/{user}/items/{item}/decrement", p.mkIncrementAPI(-1))
	r.HandleFunc("/api", p.handleCounterAPI)
	r.HandleFunc("/", p.handleCounter)
	p.b.RegisterWebName(r, "/counter", "Counter")
}

func (p *CounterPlugin) mkIncrementByNAPI(direction int) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		userName := chi.URLParam(r, "user")
		itemName := chi.URLParam(r, "item")
		delta, _ := strconv.Atoi(chi.URLParam(r, "delta"))

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
		id := ""
		u, err := p.b.DefaultConnector().Profile(userName)
		if err == nil {
			id = u.ID
		}

		item, err := GetUserItem(p.db, userName, id, itemName)
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

		body, _ := ioutil.ReadAll(r.Body)
		postData := map[string]string{}
		err = json.Unmarshal(body, &postData)
		personalMsg := ""
		if inputMsg, ok := postData["message"]; ok {
			personalMsg = fmt.Sprintf("\nMessage: %s", inputMsg)
		}

		chs := p.cfg.GetArray("channels", []string{p.cfg.Get("channels", "none")})
		req := &bot.Request{
			Conn: p.b.DefaultConnector(),
			Kind: bot.Message,
			Msg: msg.Message{
				User: &u,
				// Noting here that we're only going to do goals in a "default"
				// channel even if it should send updates to others.
				Channel: chs[0],
				Body:    fmt.Sprintf("%s += %d", itemName, delta),
				Time:    time.Now(),
			},
			Values: nil,
			Args:   nil,
		}
		msg := fmt.Sprintf("%s changed their %s counter by %d for a total of %d via the amazing %s API. %s",
			userName, itemName, delta, item.Count+delta*direction, p.cfg.Get("nick", "catbase"), personalMsg)
		for _, ch := range chs {
			p.b.Send(p.b.DefaultConnector(), bot.Message, ch, msg)
			req.Msg.Channel = ch
		}
		item.UpdateDelta(req, delta*direction)
		j, _ := json.Marshal(struct{ Status bool }{true})
		fmt.Fprint(w, string(j))
	}
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
		id := ""
		u, err := p.b.DefaultConnector().Profile(userName)
		if err == nil {
			id = u.ID
		}

		item, err := GetUserItem(p.db, userName, id, itemName)
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

		body, _ := ioutil.ReadAll(r.Body)
		postData := map[string]string{}
		err = json.Unmarshal(body, &postData)
		personalMsg := ""
		if inputMsg, ok := postData["message"]; ok {
			personalMsg = fmt.Sprintf("\nMessage: %s", inputMsg)
		}

		chs := p.cfg.GetArray("channels", []string{p.cfg.Get("channels", "none")})
		req := &bot.Request{
			Conn: p.b.DefaultConnector(),
			Kind: bot.Message,
			Msg: msg.Message{
				User: &u,
				// Noting here that we're only going to do goals in a "default"
				// channel even if it should send updates to others.
				Channel: chs[0],
				Body:    fmt.Sprintf("%s += %d", itemName, delta),
				Time:    time.Now(),
			},
			Values: nil,
			Args:   nil,
		}
		msg := fmt.Sprintf("%s changed their %s counter by %d for a total of %d via the amazing %s API. %s",
			userName, itemName, delta, item.Count+delta, p.cfg.Get("nick", "catbase"), personalMsg)
		for _, ch := range chs {
			p.b.Send(p.b.DefaultConnector(), bot.Message, ch, msg)
			req.Msg.Channel = ch
		}
		item.UpdateDelta(req, delta)
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
		req := bot.Request{
			Conn: p.b.DefaultConnector(),
			Kind: bot.Message,
			Msg: msg.Message{
				User: &user.User{
					ID:    "",
					Name:  info.User,
					Admin: false,
				},
			},
		}
		// resolveUser requires a "full" request object so we are faking it
		nick, id := p.resolveUser(req, info.User)
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
