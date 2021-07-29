package secrets

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/config"
)

//go:embed *.html
var embeddedFS embed.FS

type SecretsPlugin struct {
	b  bot.Bot
	c  *config.Config
	db *sqlx.DB
}

func New(b bot.Bot) *SecretsPlugin {
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
	r.HandleFunc("/add", p.handleRegister)
	r.HandleFunc("/remove", p.handleRemove)
	r.HandleFunc("/all", p.handleAll)
	r.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		value := r.URL.Query().Get("test")
		j, _ := json.Marshal(map[string]string{"value": value})
		w.Write(j)
	})
	r.HandleFunc("/", p.handleIndex)
	p.b.RegisterWebName(r, "/secrets", "Secrets")
}

func (p *SecretsPlugin) registerSecret(key, value string) error {
	q := `insert into secrets (key, value) values (?, ?)`
	_, err := p.db.Exec(q, key, value)
	if err != nil {
		return err
	}
	return p.c.RefreshSecrets()
}

func (p *SecretsPlugin) removeSecret(key string) error {
	q := `delete from secrets where key=?`
	_, err := p.db.Exec(q, key)
	if err != nil {
		return err
	}
	return p.c.RefreshSecrets()
}

func (p *SecretsPlugin) updateSecret(key, value string) error {
	q := `update secrets set value=? where key=?`
	_, err := p.db.Exec(q, value, key)
	if err != nil {
		return err
	}
	return p.c.RefreshSecrets()
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

func (p *SecretsPlugin) sendKeys(w http.ResponseWriter, r *http.Request) {
	checkError := mkCheckError(w)
	log.Debug().Msgf("Keys before refresh: %v", p.c.SecretKeys())
	err := p.c.RefreshSecrets()
	log.Debug().Msgf("Keys after refresh: %v", p.c.SecretKeys())
	if checkError(err) {
		return
	}
	keys, err := json.Marshal(p.c.SecretKeys())
	if checkError(err) {
		return
	}
	w.WriteHeader(200)
	w.Write(keys)
}

func (p *SecretsPlugin) handleAll(w http.ResponseWriter, r *http.Request) {
	p.sendKeys(w, r)
}

func (p *SecretsPlugin) handleRegister(w http.ResponseWriter, r *http.Request) {
	log.Debug().Msgf("handleRegister")
	if checkMethod(http.MethodPost, w, r) {
		log.Debug().Msgf("failed post %s", r.Method)
		return
	}
	checkError := mkCheckError(w)
	decoder := json.NewDecoder(r.Body)
	secret := config.Secret{}
	err := decoder.Decode(&secret)
	log.Debug().Msgf("decoding: %s", err)
	if checkError(err) {
		return
	}
	log.Debug().Msgf("Secret: %s", secret)
	err = p.registerSecret(secret.Key, secret.Value)
	if checkError(err) {
		return
	}
	p.sendKeys(w, r)
}

func (p *SecretsPlugin) handleRemove(w http.ResponseWriter, r *http.Request) {
	if checkMethod(http.MethodDelete, w, r) {
		return
	}
	checkError := mkCheckError(w)
	decoder := json.NewDecoder(r.Body)
	secret := config.Secret{}
	err := decoder.Decode(&secret)
	if checkError(err) {
		return
	}
	err = p.removeSecret(secret.Key)
	if checkError(err) {
		return
	}
	p.sendKeys(w, r)
}

func (p *SecretsPlugin) handleIndex(w http.ResponseWriter, r *http.Request) {
	index, _ := embeddedFS.ReadFile("index.html")
	w.Write(index)
}
