package counter

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
)

func (p *CounterPlugin) registerWeb() {
	r := chi.NewRouter()
	requests := p.cfg.GetInt("counter.requestsPer", 1)
	seconds := p.cfg.GetInt("counter.seconds", 1)
	dur := time.Duration(seconds) * time.Second
	subrouter := chi.NewRouter()
	subrouter.Use(httprate.LimitByIP(requests, dur))
	subrouter.HandleFunc("/api/users/{user}/items/{item}/increment/{delta}", p.mkIncrementByNAPI(1))
	subrouter.HandleFunc("/api/users/{user}/items/{item}/decrement/{delta}", p.mkIncrementByNAPI(-1))
	subrouter.HandleFunc("/api/users/{user}/items/{item}/increment", p.mkIncrementByNAPI(1))
	subrouter.HandleFunc("/api/users/{user}/items/{item}/decrement", p.mkIncrementByNAPI(-1))
	r.Mount("/", subrouter)
	r.HandleFunc("/users/{user}/items/{item}/increment", p.incHandler(1))
	r.HandleFunc("/users/{user}/items/{item}/decrement", p.incHandler(-1))
	r.HandleFunc("/", p.handleCounter)
	p.b.GetWeb().RegisterWebName(r, "/counter", "Counter")
}

func (p *CounterPlugin) incHandler(delta int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userName, _ := url.QueryUnescape(chi.URLParam(r, "user"))
		itemName, _ := url.QueryUnescape(chi.URLParam(r, "item"))
		pass := r.FormValue("password")
		if !p.b.CheckPassword("", pass) {
			w.WriteHeader(401)
			fmt.Fprintf(w, "error")
			return
		}
		item, err := p.delta(userName, itemName, "", delta)
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "error")
			return
		}
		p.renderItem(userName, item).Render(r.Context(), w)
	}
}

func (p *CounterPlugin) delta(userName, itemName, personalMessage string, delta int) (Item, error) {
	// Try to find an ID if possible
	id := ""
	u, err := p.b.DefaultConnector().Profile(userName)
	if err == nil {
		id = u.ID
	}

	item, err := GetUserItem(p.db, userName, id, itemName)
	if err != nil {
		return item, err
	}

	chs := p.cfg.GetMap("counter.channelItems", map[string]string{})
	ch, ok := chs[itemName]
	req := &bot.Request{
		Conn: p.b.DefaultConnector(),
		Kind: bot.Message,
		Msg: msg.Message{
			User: &u,
			// Noting here that we're only going to do goals in a "default"
			// channel even if it should send updates to others.
			Channel: ch,
			Body:    fmt.Sprintf("%s += %d", itemName, delta),
			Time:    time.Now(),
		},
		Values: nil,
		Args:   nil,
	}
	msg := fmt.Sprintf("%s changed their %s counter by %d for a total of %d via the amazing %s API. %s",
		userName, itemName, delta, item.Count+delta, p.cfg.Get("nick", "catbase"), personalMessage)
	if !ok {
		chs := p.cfg.GetArray("counter.channels", []string{})
		for _, ch := range chs {
			p.b.Send(p.b.DefaultConnector(), bot.Message, ch, msg)
			req.Msg.Channel = ch
		}
	} else {
		p.b.Send(p.b.DefaultConnector(), bot.Message, ch, msg)
		req.Msg.Channel = ch
	}
	if err := item.UpdateDelta(req, delta); err != nil {
		return item, err
	}
	return item, nil
}

func (p *CounterPlugin) mkIncrementByNAPI(direction int) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		userName, _ := url.QueryUnescape(chi.URLParam(r, "user"))
		itemName, _ := url.QueryUnescape(chi.URLParam(r, "item"))
		delta, err := strconv.Atoi(chi.URLParam(r, "delta"))
		if err != nil || delta == 0 {
			delta = direction
		} else {
			delta = delta * direction
		}

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

		body, _ := io.ReadAll(r.Body)
		postData := map[string]string{}
		err = json.Unmarshal(body, &postData)
		personalMsg := ""
		if inputMsg, ok := postData["message"]; ok {
			personalMsg = fmt.Sprintf("\nMessage: %s", inputMsg)
		}

		if _, err := p.delta(userName, itemName, personalMsg, delta*direction); err != nil {
			log.Error().Err(err).Msg("error finding item")
			w.WriteHeader(400)
			j, _ := json.Marshal(struct {
				Status bool
				Error  error
			}{false, err})
			fmt.Fprint(w, string(j))
			return
		}

		j, _ := json.Marshal(struct{ Status bool }{true})
		fmt.Fprint(w, string(j))
	}
}

func (p *CounterPlugin) handleCounter(w http.ResponseWriter, r *http.Request) {
	p.b.GetWeb().Index("Counter", p.index()).Render(r.Context(), w)
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
