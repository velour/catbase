package cowboy

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (p *Cowboy) registerWeb() {
	r := chi.NewRouter()
	r.HandleFunc("/img/{what}", p.handleImage)
	p.b.RegisterWeb(r, "/cowboy")
}

func (p *Cowboy) handleImage(w http.ResponseWriter, r *http.Request) {
	what := chi.URLParam(r, "what")
	img, err := encode(cowboy(p.c, p.emojyPath, p.baseEmojyURL, what))
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Error: %s", err)
		return
	}
	w.Header().Add("Content-Type", "image/png")
	w.Write(img)
}
