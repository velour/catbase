package admin

import (
	"crypto/md5"
	"crypto/rand"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

//go:embed *.html
var embeddedFS embed.FS

func (p *AdminPlugin) registerWeb() {
	r := chi.NewRouter()
	r.HandleFunc("/api", p.handleVarsAPI)
	r.HandleFunc("/", p.handleVars)
	p.bot.RegisterWebName(r, "/vars", "Variables")
	r = chi.NewRouter()
	r.HandleFunc("/verify", p.handleAppPassCheck)
	r.HandleFunc("/api", p.handleAppPassAPI)
	r.HandleFunc("/", p.handleAppPass)
	p.bot.RegisterWebName(r, "/apppass", "App Pass")
}

func (p *AdminPlugin) handleAppPass(w http.ResponseWriter, r *http.Request) {
	index, _ := embeddedFS.ReadFile("apppass.html")
	w.Write(index)
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
	req := struct {
		Password  string    `json:"password"`
		PassEntry PassEntry `json:"passEntry"`
	}{}
	body, _ := ioutil.ReadAll(r.Body)
	_ = json.Unmarshal(body, &req)
	if req.PassEntry.Secret == "" {
		writeErr(w, fmt.Errorf("missing secret"))
		return
	}
	if req.Password == "" || !p.bot.CheckPassword(req.PassEntry.Secret, req.Password) {
		writeErr(w, fmt.Errorf("missing or incorrect password"))
		return
	}
	switch r.Method {
	case http.MethodPut:
		if req.PassEntry.Secret == "" {
			writeErr(w, fmt.Errorf("missing secret"))
			return
		}
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
			writeErr(w, err)
			return
		}
		id, err := res.LastInsertId()
		if err != nil {
			writeErr(w, err)
			return
		}
		req.PassEntry.ID = id
		j, _ := json.Marshal(req.PassEntry)
		fmt.Fprint(w, string(j))
		return
	case http.MethodDelete:
		if req.PassEntry.ID <= 0 {
			writeErr(w, fmt.Errorf("missing ID"))
			return
		}
		q := `delete from apppass where id = ?`
		_, err := p.db.Exec(q, req.PassEntry.ID)
		if err != nil {
			writeErr(w, err)
			return
		}
	}
	q := `select id,secret from apppass where secret = ?`
	passEntries := []PassEntry{}
	err := p.db.Select(&passEntries, q, req.PassEntry.Secret)
	if err != nil {
		writeErr(w, err)
		return
	}
	j, _ := json.Marshal(passEntries)
	_, _ = fmt.Fprint(w, string(j))
}

func writeErr(w http.ResponseWriter, err error) {
	log.Error().Err(err).Msg("apppass error")
	j, _ := json.Marshal(struct {
		Err string `json:"err"`
	}{
		err.Error(),
	})
	w.WriteHeader(400)
	fmt.Fprint(w, string(j))
}

type configEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (p *AdminPlugin) handleVars(w http.ResponseWriter, r *http.Request) {
	tpl := template.Must(template.ParseFS(embeddedFS, "vars.html"))
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

	if err := tpl.Execute(w, struct {
		Items []configEntry
	}{configEntries}); err != nil {
		log.Error().Err(err).Msg("template error")
		w.WriteHeader(500)
		fmt.Fprint(w, "Error parsing template")
	}
}

func (p *AdminPlugin) handleVarsAPI(w http.ResponseWriter, r *http.Request) {
	var configEntries []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
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
	for i, e := range configEntries {
		if strings.Contains(e.Value, ";;") {
			e.Value = strings.ReplaceAll(e.Value, ";;", ", ")
			e.Value = fmt.Sprintf("[%s]", e.Value)
			configEntries[i] = e
		}
	}
	j, _ := json.Marshal(configEntries)
	fmt.Fprintf(w, "%s", j)
}
