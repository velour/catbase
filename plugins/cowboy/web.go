package cowboy

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (p *Cowboy) registerWeb() {
	r := chi.NewRouter()
	r.HandleFunc("/img/{overlay}/{what}", p.handleImage)
	p.b.GetWeb().RegisterWeb(r, "/cowboy")
}

func (p *Cowboy) handleImage(w http.ResponseWriter, r *http.Request) {
	what := chi.URLParam(r, "what")
	overlay := chi.URLParam(r, "overlay")
	overlays := p.c.GetMap("cowboy.overlays", defaultOverlays)
	overlayPath, ok := overlays[overlay]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	img, err := encode(cowboy(p.emojyPath, p.baseEmojyURL, overlayPath, what))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error: %s", err)
		return
	}
	w.Header().Add("Content-Type", "image/png")
	w.Write(img)
}
