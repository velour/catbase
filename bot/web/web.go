package web

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/bot/stats"
	"github.com/velour/catbase/config"
	"net/http"
	"strings"
	"time"

	httpin_integration "github.com/ggicci/httpin/integration"
)

type Web struct {
	config        *config.Config
	router        *chi.Mux
	httpEndPoints []EndPoint
	stats         *stats.Stats
}

type EndPoint struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// GetWebNavigation returns a list of bootstrap-vue <b-nav-item> links
// The parent <nav> is not included so each page may display it as
// best fits
func (ws *Web) GetWebNavigation() []EndPoint {
	endpoints := ws.httpEndPoints
	moreEndpoints := ws.config.GetArray("bot.links", []string{})
	for _, e := range moreEndpoints {
		link := strings.SplitN(e, ":", 2)
		if len(link) != 2 {
			continue
		}
		endpoints = append(endpoints, EndPoint{link[0], link[1]})
	}
	return endpoints
}

func (ws *Web) serveRoot(w http.ResponseWriter, r *http.Request) {
	ws.Index("Home", ws.showStats()).Render(r.Context(), w)
}

func (ws *Web) serveNavHTML(w http.ResponseWriter, r *http.Request) {
	currentPage := chi.URLParam(r, "currentPage")
	ws.Nav(currentPage).Render(r.Context(), w)
}

func (ws *Web) serveNav(w http.ResponseWriter, r *http.Request) {
	enc := json.NewEncoder(w)
	err := enc.Encode(ws.GetWebNavigation())
	if err != nil {
		jsonErr, _ := json.Marshal(err)
		w.WriteHeader(500)
		w.Write(jsonErr)
	}
}

func (ws *Web) setupHTTP() {
	// Make the http logger optional
	// It has never served a purpose in production and with the emojy page, can make a rather noisy log
	if ws.config.GetInt("bot.useLogger", 0) == 1 {
		ws.router.Use(middleware.Logger)
	}

	reqCount := ws.config.GetInt("bot.httprate.requests", 500)
	reqTime := time.Duration(ws.config.GetInt("bot.httprate.seconds", 5))
	if reqCount > 0 && reqTime > 0 {
		ws.router.Use(httprate.LimitByIP(reqCount, reqTime*time.Second))
	}

	ws.router.Use(middleware.RequestID)
	ws.router.Use(middleware.Recoverer)
	ws.router.Use(middleware.StripSlashes)

	httpin_integration.UseGochiURLParam("path", chi.URLParam)

	ws.router.HandleFunc("/", ws.serveRoot)
	ws.router.HandleFunc("/nav", ws.serveNav)
	ws.router.HandleFunc("/navHTML", ws.serveNavHTML)
	ws.router.HandleFunc("/navHTML/{currentPage}", ws.serveNavHTML)
}

func (ws *Web) RegisterWeb(r http.Handler, root string) {
	ws.router.Mount(root, r)
}

func (ws *Web) RegisterWebName(r http.Handler, root, name string) {
	ws.httpEndPoints = append(ws.httpEndPoints, EndPoint{name, root})
	ws.router.Mount(root, r)
}

func (ws *Web) ListenAndServe(addr string) {
	log.Debug().Msgf("starting web service at %s", addr)
	log.Fatal().Err(http.ListenAndServe(addr, ws.router)).Msg("bot killed")
}

func New(config *config.Config, s *stats.Stats) *Web {
	w := &Web{
		config: config,
		router: chi.NewRouter(),
		stats:  s,
	}
	w.setupHTTP()
	return w
}

func (ws *Web) botName() string {
	return ws.config.Get("nick", "catbase")
}
