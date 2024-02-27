package bot

import (
	"embed"
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"net/http"
	"strings"
)

//go:embed *.html
var embeddedFS embed.FS

func (b *bot) serveRoot(w http.ResponseWriter, r *http.Request) {
	b.index().Render(r.Context(), w)
}

func (b *bot) serveNavHTML(w http.ResponseWriter, r *http.Request) {
	currentPage := chi.URLParam(r, "currentPage")
	b.Nav(currentPage, b.GetWebNavigation()).Render(r.Context(), w)
}

func (b *bot) serveNav(w http.ResponseWriter, r *http.Request) {
	enc := json.NewEncoder(w)
	err := enc.Encode(b.GetWebNavigation())
	if err != nil {
		jsonErr, _ := json.Marshal(err)
		w.WriteHeader(500)
		w.Write(jsonErr)
	}
}

// GetWebNavigation returns a list of bootstrap-vue <b-nav-item> links
// The parent <nav> is not included so each page may display it as
// best fits
func (b *bot) GetWebNavigation() []EndPoint {
	endpoints := b.httpEndPoints
	moreEndpoints := b.config.GetArray("bot.links", []string{})
	for _, e := range moreEndpoints {
		link := strings.SplitN(e, ":", 2)
		if len(link) != 2 {
			continue
		}
		endpoints = append(endpoints, EndPoint{link[0], link[1]})
	}
	return endpoints
}
