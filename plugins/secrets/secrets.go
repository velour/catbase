package secrets

import (
	"encoding/json"
	"fmt"
	"github.com/ggicci/httpin"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/config"
	"io"
	"net/http"
	"net/url"
)

type SecretsPlugin struct {
	b  bot.Bot
	c  *config.Config
	db *sqlx.DB
}

func New(b bot.Bot) bot.Plugin {
	p := &SecretsPlugin{
		b:  b,
		c:  b.Config(),
		db: b.DB(),
	}
	p.registerWeb()
	return p
}

func (p *SecretsPlugin) registerWeb() {
	r := chi.NewRouter()
	r.With(httpin.NewInput(RegisterReq{})).
		Post("/add", p.handleRegister)
	r.Delete("/remove", p.handleRemove)
	r.Get("/", p.handleIndex)
	p.b.GetWeb().RegisterWebName(r, "/secrets", "Secrets")
}

func mkCheckError(w http.ResponseWriter) func(error) bool {
	return func(err error) bool {
		if err != nil {
			log.Error().Stack().Err(err).Msgf("secret failed")
			w.WriteHeader(500)
			e, _ := json.Marshal(err)
			w.Write(e)
			return true
		}
		return false
	}
}

func checkMethod(method string, w http.ResponseWriter, r *http.Request) bool {
	if r.Method != method {
		w.WriteHeader(405)
		fmt.Fprintf(w, "Incorrect HTTP method")
		return true
	}
	return false
}

func (p *SecretsPlugin) keys() []string {
	return p.c.SecretKeys()
}

func (p *SecretsPlugin) sendKeys(w http.ResponseWriter, r *http.Request) {
	p.keysList().Render(r.Context(), w)
}

func (p *SecretsPlugin) handleAll(w http.ResponseWriter, r *http.Request) {
	p.sendKeys(w, r)
}

type RegisterReq struct {
	Key   string `in:"form=key"`
	Value string `in:"form=value"`
}

func (p *SecretsPlugin) handleRegister(w http.ResponseWriter, r *http.Request) {
	input := r.Context().Value(httpin.Input).(*RegisterReq)
	checkError := mkCheckError(w)
	err := p.c.RegisterSecret(input.Key, input.Value)
	if checkError(err) {
		return
	}
	p.sendKeys(w, r)
}

func (p *SecretsPlugin) handleRemove(w http.ResponseWriter, r *http.Request) {
	checkError := mkCheckError(w)
	b, err := io.ReadAll(r.Body)
	if checkError(err) {
		return
	}
	q, err := url.ParseQuery(string(b))
	if checkError(err) {
		return
	}
	secret := q.Get("key")
	err = p.c.RemoveSecret(secret)
	if checkError(err) {
		return
	}
	p.sendKeys(w, r)
}

func (p *SecretsPlugin) handleIndex(w http.ResponseWriter, r *http.Request) {
	p.b.GetWeb().Index("Secrets", p.index()).Render(r.Context(), w)
}
