package tappd

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"net/http"
)

func (p *Tappd) registerWeb() {
	r := chi.NewRouter()
	r.HandleFunc("/", p.serveImage)
	p.b.RegisterWeb(r, "/tappd/{id}")
}

func (p *Tappd) serveImage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	imgData, ok := p.imageMap[id]
	log.Debug().
		Str("id", id).
		Str("SrcURL", imgData.SrcURL).
		Bool("ok", ok).
		Msgf("creating request")
	if !ok {
		w.WriteHeader(404)
		out, _ := json.Marshal(struct{ err string }{"could not find ID"})
		w.Write(out)
		return
	}
	w.Write(imgData.Repr)
}
