package emojy

import (
	"embed"
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"net/http"
)

//go:embed *.html
var embeddedFS embed.FS

func (p *EmojyPlugin) registerWeb() {
	r := chi.NewRouter()
	r.HandleFunc("/all", p.handleAll)
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
