package admin

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"fmt"
	"github.com/ggicci/httpin"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

func (p *AdminPlugin) registerWeb() {
	r := chi.NewRouter()
	r.HandleFunc("/", p.handleVars)
	p.bot.GetWeb().RegisterWebName(r, "/vars", "Variables")
	r = chi.NewRouter()
	r.With(httpin.NewInput(AppPassCheckReq{})).
		HandleFunc("/verify", p.handleAppPassCheck)
	//r.HandleFunc("/api", p.handleAppPassAPI)
	r.With(httpin.NewInput(AppPassAPIReq{})).
		Put("/api", p.handleAppPassAPIPut)
	r.With(httpin.NewInput(AppPassAPIReq{})).
		Delete("/api", p.handleAppPassAPIDelete)
	r.With(httpin.NewInput(AppPassAPIReq{})).
		Post("/api", p.handleAppPassAPIPost)
	r.HandleFunc("/", p.handleAppPass)
	p.bot.GetWeb().RegisterWebName(r, "/apppass", "App Pass")
}

func (p *AdminPlugin) handleAppPass(w http.ResponseWriter, r *http.Request) {
	p.bot.GetWeb().Index("App Pass", p.page()).Render(r.Context(), w)
}

type PassEntry struct {
	ID     int64  `json:"id" in:"form=id;query=password"`
	Secret string `json:"secret" in:"form=secret;query=password"`

	// Should be null unless inserting a new entry
	Pass        string `json:"pass"`
	encodedPass []byte `json:"encodedPass"`
	Cost        int    `json:"cost"`
}

func (p *PassEntry) EncodePass() {
	encoded, err := bcrypt.GenerateFromPassword([]byte(p.Pass), p.Cost)
	if err != nil {
		log.Error().Err(err).Msg("could not hash password")
	}
	p.encodedPass = encoded
}

func (p *PassEntry) Compare(pass string) bool {
	if err := bcrypt.CompareHashAndPassword(p.encodedPass, []byte(pass)); err != nil {
		log.Error().Err(err).Msg("failure to match password")
		return false
	}
	return true
}

type AppPassCheckReq struct {
	Secret   string `json:"secret"`
	Password string `json:"password"`
}

func (p *AdminPlugin) handleAppPassCheck(w http.ResponseWriter, r *http.Request) {
	input := r.Context().Value(httpin.Input).(*AppPassCheckReq)
	if p.bot.CheckPassword(input.Secret, input.Password) {
		w.WriteHeader(204)
	} else {
		w.WriteHeader(403)
	}
}

type AppPassAPIReq struct {
	Password  string `in:"form=password;query=password"`
	PassEntry PassEntry
}

func (p *AdminPlugin) checkAPIInput(w http.ResponseWriter, r *http.Request) *AppPassAPIReq {
	input := r.Context().Value(httpin.Input).(*AppPassAPIReq)

	if input.Password == "" && input.PassEntry.Secret == "" {
		b, _ := io.ReadAll(r.Body)
		query, _ := url.ParseQuery(string(b))
		input.Password = query.Get("password")
		input.PassEntry.ID, _ = strconv.ParseInt(query.Get("id"), 10, 64)
		input.PassEntry.Secret = query.Get("secret")
	}

	log.Printf("checkAPIInput: %#v", input)

	if input.PassEntry.Secret == "" {
		writeErr(r.Context(), w, fmt.Errorf("missing secret"))
		return nil
	}
	if input.Password == "" || !p.bot.CheckPassword(input.PassEntry.Secret, input.Password) {
		writeErr(r.Context(), w, fmt.Errorf("missing or incorrect password"))
		return nil
	}
	return input
}

func (p *AdminPlugin) handleAppPassAPIPut(w http.ResponseWriter, r *http.Request) {
	input := p.checkAPIInput(w, r)
	if input == nil {
		return
	}
	if string(input.PassEntry.Pass) == "" {
		c := 10
		b := make([]byte, c)
		_, err := rand.Read(b)
		if err != nil {
			fmt.Println("error:", err)
			return
		}
		input.PassEntry.Pass = fmt.Sprintf("%x", md5.Sum(b))
	}
	q := `insert into apppass (secret, encoded_pass, cost) values (?, ?, ?)`
	input.PassEntry.EncodePass()

	check := bcrypt.CompareHashAndPassword(input.PassEntry.encodedPass, []byte(input.PassEntry.Pass))

	log.Debug().
		Str("secret", input.PassEntry.Secret).
		Str("encoded", string(input.PassEntry.encodedPass)).
		Str("password", string(input.PassEntry.Pass)).
		Interface("check", check).
		Msg("debug pass creation")

	res, err := p.db.Exec(q, input.PassEntry.Secret, input.PassEntry.encodedPass, input.PassEntry.Cost)
	if err != nil {
		writeErr(r.Context(), w, err)
		return
	}
	id, err := res.LastInsertId()
	if err != nil {
		writeErr(r.Context(), w, err)
		return
	}
	input.PassEntry.ID = id
	p.showPassword(input.PassEntry).Render(r.Context(), w)
}

func (p *AdminPlugin) handleAppPassAPIDelete(w http.ResponseWriter, r *http.Request) {
	input := p.checkAPIInput(w, r)
	if input == nil {
		return
	}
	if input.PassEntry.ID <= 0 {
		writeErr(r.Context(), w, fmt.Errorf("missing ID"))
		return
	}
	q := `delete from apppass where id = ?`
	_, err := p.db.Exec(q, input.PassEntry.ID)
	if err != nil {
		writeErr(r.Context(), w, err)
		return
	}
	p.handleAppPassAPIPost(w, r)
}

func (p *AdminPlugin) handleAppPassAPIPost(w http.ResponseWriter, r *http.Request) {
	input := p.checkAPIInput(w, r)
	if input == nil {
		return
	}
	q := `select id,secret from apppass where secret = ?`
	passEntries := []PassEntry{}

	err := p.db.Select(&passEntries, q, input.PassEntry.Secret)
	if err != nil {
		writeErr(r.Context(), w, err)
		return
	}
	p.entries(passEntries).Render(r.Context(), w)
}

func writeErr(ctx context.Context, w http.ResponseWriter, err error) {
	log.Error().Err(err).Msg("apppass error")
	renderError(err).Render(ctx, w)
}

type configEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (p *AdminPlugin) handleVars(w http.ResponseWriter, r *http.Request) {
	var configEntries []configEntry
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

	p.bot.GetWeb().Index("Variables", vars(configEntries)).Render(r.Context(), w)
}
