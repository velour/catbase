package bot

import (
	"embed"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"net/http"
	"strings"
	"text/template"
)

//go:embed *.html
var embeddedFS embed.FS

func (b *bot) serveRoot(w http.ResponseWriter, r *http.Request) {
	index, _ := embeddedFS.ReadFile("index.html")
	w.Write(index)
}

func (b *bot) serveNavHTML(w http.ResponseWriter, r *http.Request) {
	currentPage := chi.URLParam(r, "currentPage")
	tpl := template.Must(template.ParseFS(embeddedFS, "nav.html"))
	if err := tpl.Execute(w, struct {
		CurrentPage string
		Items       []EndPoint
	}{currentPage, b.GetWebNavigation()}); err != nil {
		log.Error().Err(err).Msg("template error")
		w.WriteHeader(500)
		fmt.Fprint(w, "Error parsing nav template")
	}
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
