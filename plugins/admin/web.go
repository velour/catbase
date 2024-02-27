package admin

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
)

func (p *AdminPlugin) registerWeb() {
	r := chi.NewRouter()
	r.HandleFunc("/", p.handleVars)
	p.bot.GetWeb().RegisterWebName(r, "/vars", "Variables")
	r = chi.NewRouter()
	r.HandleFunc("/verify", p.handleAppPassCheck)
	r.HandleFunc("/api", p.handleAppPassAPI)
	r.HandleFunc("/", p.handleAppPass)
	p.bot.GetWeb().RegisterWebName(r, "/apppass", "App Pass")
}

func (p *AdminPlugin) handleAppPass(w http.ResponseWriter, r *http.Request) {
	p.bot.GetWeb().Index("App Pass", p.page()).Render(r.Context(), w)
}

type PassEntry struct {
	ID     int64  `json:"id"`
	Secret string `json:"secret"`

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

func (p *AdminPlugin) handleAppPassCheck(w http.ResponseWriter, r *http.Request) {
	req := struct {
		Secret   string `json:"secret"`
		Password string `json:"password"`
	}{}
	body, _ := ioutil.ReadAll(r.Body)
	_ = json.Unmarshal(body, &req)
	if p.bot.CheckPassword(req.Secret, req.Password) {
		w.WriteHeader(204)
	} else {
		w.WriteHeader(403)
	}
}

func (p *AdminPlugin) handleAppPassAPI(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	b, _ := io.ReadAll(r.Body)
	query, _ := url.ParseQuery(string(b))
	secret := r.FormValue("secret")
	password := r.FormValue("password")
	id, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if !r.Form.Has("secret") && query.Has("secret") {
		secret = query.Get("secret")
	}
	if !r.Form.Has("password") && query.Has("password") {
		password = query.Get("password")
	}
	if !r.Form.Has("id") && query.Has("id") {
		id, _ = strconv.ParseInt(query.Get("id"), 10, 64)
	}
	req := struct {
		Password  string    `json:"password"`
		PassEntry PassEntry `json:"passEntry"`
	}{
		password,
		PassEntry{
			ID:     id,
			Secret: secret,
		},
	}
	if req.PassEntry.Secret == "" {
		writeErr(r.Context(), w, fmt.Errorf("missing secret"))
		return
	}
	if req.Password == "" || !p.bot.CheckPassword(req.PassEntry.Secret, req.Password) {
		writeErr(r.Context(), w, fmt.Errorf("missing or incorrect password"))
		return
	}
	switch r.Method {
	case http.MethodPut:
		if string(req.PassEntry.Pass) == "" {
			c := 10
			b := make([]byte, c)
			_, err := rand.Read(b)
			if err != nil {
				fmt.Println("error:", err)
				return
			}
			req.PassEntry.Pass = fmt.Sprintf("%x", md5.Sum(b))
		}
		q := `insert into apppass (secret, encoded_pass, cost) values (?, ?, ?)`
		req.PassEntry.EncodePass()

		check := bcrypt.CompareHashAndPassword(req.PassEntry.encodedPass, []byte(req.PassEntry.Pass))

		log.Debug().
			Str("secret", req.PassEntry.Secret).
			Str("encoded", string(req.PassEntry.encodedPass)).
			Str("password", string(req.PassEntry.Pass)).
			Interface("check", check).
			Msg("debug pass creation")

		res, err := p.db.Exec(q, req.PassEntry.Secret, req.PassEntry.encodedPass, req.PassEntry.Cost)
		if err != nil {
			writeErr(r.Context(), w, err)
			return
		}
		id, err := res.LastInsertId()
		if err != nil {
			writeErr(r.Context(), w, err)
			return
		}
		req.PassEntry.ID = id
		p.showPassword(req.PassEntry).Render(r.Context(), w)
		return
	case http.MethodDelete:

		if req.PassEntry.ID <= 0 {
			writeErr(r.Context(), w, fmt.Errorf("missing ID"))
			return
		}
		q := `delete from apppass where id = ?`
		_, err := p.db.Exec(q, req.PassEntry.ID)
		if err != nil {
			writeErr(r.Context(), w, err)
			return
		}
	}
	q := `select id,secret from apppass where secret = ?`
	passEntries := []PassEntry{}
	err := p.db.Select(&passEntries, q, req.PassEntry.Secret)
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
