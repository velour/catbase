package counter

import (
	"encoding/json"
	"fmt"
	"github.com/ggicci/httpin"
	"net/http"
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
	subrouter.With(httpin.NewInput(CounterChangeReq{})).
		HandleFunc("/api/users/{user}/items/{item}/increment/{delta}", p.mkIncrementByNAPI(1))
	subrouter.With(httpin.NewInput(CounterChangeReq{})).
		HandleFunc("/api/users/{user}/items/{item}/decrement/{delta}", p.mkIncrementByNAPI(-1))
	subrouter.With(httpin.NewInput(CounterChangeReq{})).
		HandleFunc("/api/users/{user}/items/{item}/increment", p.mkIncrementByNAPI(1))
	subrouter.With(httpin.NewInput(CounterChangeReq{})).
		HandleFunc("/api/users/{user}/items/{item}/decrement", p.mkIncrementByNAPI(-1))
	r.Mount("/", subrouter)
	r.With(httpin.NewInput(CounterChangeReq{})).
		HandleFunc("/users/{user}/items/{item}/increment", p.incHandler(1))
	r.With(httpin.NewInput(CounterChangeReq{})).
		HandleFunc("/users/{user}/items/{item}/decrement", p.incHandler(-1))
	r.HandleFunc("/", p.handleCounter)
	p.b.GetWeb().RegisterWebName(r, "/counter", "Counter")
}

type CounterChangeReq struct {
	UserName string `in:"path=user"`
	Item     string `in:"path=item"`
	Password string `in:"form=password"`
	Delta    int64  `in:"path=delta"`
	Body     struct {
		Message string `json:"message"`
	} `in:"body=json"`
}

func (p *CounterPlugin) incHandler(delta int64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		input := r.Context().Value(httpin.Input).(*CounterChangeReq)
		if !p.b.CheckPassword("", input.Password) {
			w.WriteHeader(401)
			fmt.Fprintf(w, "error")
			return
		}
		item, err := p.delta(input.UserName, input.Item, "", delta)
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "error")
			return
		}
		p.renderItem(input.UserName, item).Render(r.Context(), w)
	}
}

func (p *CounterPlugin) delta(userName, itemName, personalMessage string, delta int64) (Item, error) {
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

func (p *CounterPlugin) mkIncrementByNAPI(direction int64) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		input := r.Context().Value(httpin.Input).(*CounterChangeReq)
		if input.Delta == 0 {
			input.Delta = direction
		} else {
			input.Delta = input.Delta * direction
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

		personalMsg := ""
		if input.Body.Message != "" {
			personalMsg = fmt.Sprintf("\nMessage: %s", input.Body.Message)
		}

		if _, err := p.delta(input.UserName, input.Item, personalMsg, input.Delta*direction); err != nil {
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
	Amount int64 `json:"amount"`
}
