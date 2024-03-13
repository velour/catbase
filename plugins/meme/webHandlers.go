package meme

import (
	"encoding/json"
	"fmt"
	"github.com/ggicci/httpin"
	"net/http"
	"net/url"
	"sort"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/velour/catbase/bot"
)

func (p *MemePlugin) registerWeb(c bot.Connector) {
	r := chi.NewRouter()
	r.HandleFunc("/slash", p.slashMeme(c))
	r.Get("/img", p.img)
	r.With(httpin.NewInput(SaveReq{})).
		Put("/save/{name}", p.saveMeme)
	r.With(httpin.NewInput(SaveReq{})).
		Post("/add", p.saveMeme)
	r.Delete("/rm/{name}", p.rmMeme)
	r.Get("/edit/{name}", p.editMeme)
	r.Get("/", p.webRoot)
	p.bot.GetWeb().RegisterWebName(r, "/meme", "Memes")
}

type webResp struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Config string `json:"config"`
}

type webResps []webResp

func (w webResps) Len() int      { return len(w) }
func (w webResps) Swap(i, j int) { w[i], w[j] = w[j], w[i] }

type ByName struct{ webResps }

func (s ByName) Less(i, j int) bool { return s.webResps[i].Name < s.webResps[j].Name }

func (p *MemePlugin) all() webResps {
	memes := p.c.GetMap("meme.memes", defaultFormats)
	configs := p.c.GetMap("meme.memeconfigs", map[string]string{})

	values := webResps{}
	for n, u := range memes {
		config, ok := configs[n]
		if !ok {
			b, _ := json.MarshalIndent(p.defaultFormatConfig(), " ", " ")
			config = string(b)
		}
		realURL, err := url.Parse(u)
		if err != nil || realURL.Scheme == "" {
			values = append(values, webResp{n, "404.png", config})
			log.Error().Err(err).Msgf("invalid URL")
			continue
		}
		values = append(values, webResp{n, realURL.String(), config})
	}
	sort.Sort(ByName{values})

	return values
}

func mkCheckError(w http.ResponseWriter) func(error) bool {
	return func(err error) bool {
		if err != nil {
			log.Error().Err(err).Msgf("meme failed")
			w.WriteHeader(500)
			e, _ := json.Marshal(err)
			w.Write(e)
			return true
		}
		return false
	}
}

func (p *MemePlugin) rmMeme(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	formats := p.c.GetMap("meme.memes", defaultFormats)
	delete(formats, name)
	err := p.c.SetMap("meme.memes", formats)
	mkCheckError(w)(err)
}

type SaveReq struct {
	Name   string `in:"path=name"`
	Config string `in:"form=config"`
	URL    string `in:"form=url"`
}

func (p *MemePlugin) saveMeme(w http.ResponseWriter, r *http.Request) {
	input := r.Context().Value(httpin.Input).(*SaveReq)
	checkError := mkCheckError(w)

	formats := p.c.GetMap("meme.memes", defaultFormats)
	formats[input.Name] = input.URL
	err := p.c.SetMap("meme.memes", formats)
	checkError(err)

	if input.Config == "" {
		input.Config = p.defaultFormatConfigJSON()
	}
	configs := p.c.GetMap("meme.memeconfigs", map[string]string{})
	configs[input.Name] = input.Config
	err = p.c.SetMap("meme.memeconfigs", configs)
	checkError(err)

	meme := webResp{
		Name:   input.Name,
		URL:    formats[input.Name],
		Config: configs[input.Name],
	}

	p.Show(meme).Render(r.Context(), w)
}

func (p *MemePlugin) webRoot(w http.ResponseWriter, r *http.Request) {
	p.bot.GetWeb().Index("Meme", p.index(p.all())).Render(r.Context(), w)
}

func (p *MemePlugin) editMeme(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	memes := p.c.GetMap("meme.memes", defaultFormats)
	configs := p.c.GetMap("meme.memeconfigs", map[string]string{})
	meme, ok := memes[name]
	if !ok {
		fmt.Fprintf(w, "Didn't find that meme.")
	}
	resp := webResp{
		Name:   name,
		URL:    meme,
		Config: configs[name],
	}
	p.Edit(resp).Render(r.Context(), w)
}

func (p *MemePlugin) img(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	spec := q.Get("spec")
	if spec == "" {
		log.Debug().Msgf("No spec found for img")
		w.WriteHeader(404)
		w.Write([]byte{})
		return
	}

	s, err := SpecFromJSON([]byte(spec))
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	img, err := p.genMeme(s)

	if err == nil {
		w.Write(img)
	} else {
		log.Error().
			Err(err).
			Interface("spec", s).
			Msg("Unable to generate meme image")
		w.WriteHeader(404)
		w.Write([]byte("not found"))
	}
	p.images.cleanup()
}
