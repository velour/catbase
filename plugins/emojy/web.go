package emojy

import (
	"embed"
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"net/http"
	"path"
)

//go:embed *.html
var embeddedFS embed.FS

func (p *EmojyPlugin) registerWeb() {
	r := chi.NewRouter()
	r.HandleFunc("/all", p.handleAll)
	r.HandleFunc("/file/{name}", p.handleEmojy)
	r.HandleFunc("/", p.handleIndex)
	p.b.RegisterWebName(r, "/emojy", "Emojys")
}

func (p *EmojyPlugin) handleIndex(w http.ResponseWriter, r *http.Request) {
	index, _ := embeddedFS.ReadFile("index.html")
	w.Write(index)
}

func (p *EmojyPlugin) handleAll(w http.ResponseWriter, r *http.Request) {
	emojy, err := p.allCounts()
	if err != nil {
		w.WriteHeader(500)
		log.Error().Err(err).Msgf("handleAll")
		out, _ := json.Marshal(struct{ err error }{err})
		w.Write(out)
	}
	out, _ := json.Marshal(emojy)
	w.Write(out)
}

func (p *EmojyPlugin) handleEmojy(w http.ResponseWriter, r *http.Request) {
	fname := chi.URLParam(r, "name")
	emojyPath := p.c.Get("emojy.path", "emojy")
	contents, err := ioutil.ReadFile(path.Join(emojyPath, fname))
	if err != nil {
		w.WriteHeader(404)
		out, _ := json.Marshal(struct{ err error }{err})
		w.Write(out)
	}
	w.Write(contents)
}
