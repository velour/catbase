package meme

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"path"
	"sort"

	"github.com/rs/zerolog/log"

	"github.com/velour/catbase/bot"
)

func (p *MemePlugin) registerWeb(c bot.Connector) {
	http.HandleFunc("/slash/meme", p.slashMeme(c))
	http.HandleFunc("/meme/img/", p.img)
	http.HandleFunc("/meme/all", p.all)
	http.HandleFunc("/meme/add", p.addMeme)
	http.HandleFunc("/meme/rm", p.rmMeme)
	http.HandleFunc("/meme", p.webRoot)
	p.bot.RegisterWeb("/meme", "Memes")
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

func (p *MemePlugin) all(w http.ResponseWriter, r *http.Request) {
	memes := p.c.GetMap("meme.memes", defaultFormats)
	configs := p.c.GetMap("meme.memeconfigs", map[string]string{})

	values := webResps{}
	for n, u := range memes {
		config, ok := configs[n]
		if !ok {
			b, _ := json.Marshal(defaultFormatConfig)
			config = string(b)
		}
		realURL, err := url.Parse(u)
		if err != nil || realURL.Scheme == "" {
			realURL, err = url.Parse("https://imgflip.com/s/meme/" + u)
			if err != nil {
				values = append(values, webResp{n, "404.png", config})
				log.Error().Err(err).Msgf("invalid URL")
				continue
			}
		}
		values = append(values, webResp{n, realURL.String(), config})
	}
	sort.Sort(ByName{values})

	out, err := json.Marshal(values)
	if err != nil {
		w.WriteHeader(500)
		log.Error().Err(err).Msgf("could not serve all memes route")
		return
	}
	w.Write(out)
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
	if r.Method != http.MethodDelete {
		w.WriteHeader(405)
		fmt.Fprintf(w, "Incorrect HTTP method")
		return
	}
	checkError := mkCheckError(w)
	decoder := json.NewDecoder(r.Body)
	values := webResp{}
	err := decoder.Decode(&values)
	if checkError(err) {
		return
	}
	formats := p.c.GetMap("meme.memes", defaultFormats)
	delete(formats, values.Name)
	err = p.c.SetMap("meme.memes", formats)
	checkError(err)
}

func (p *MemePlugin) addMeme(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		fmt.Fprintf(w, "Incorrect HTTP method")
		return
	}
	checkError := mkCheckError(w)
	decoder := json.NewDecoder(r.Body)
	values := webResp{}
	err := decoder.Decode(&values)
	if checkError(err) {
		log.Error().Err(err).Msgf("could not decode body")
		return
	}
	log.Debug().Msgf("POSTed values: %+v", values)
	formats := p.c.GetMap("meme.memes", defaultFormats)
	formats[values.Name] = values.URL
	err = p.c.SetMap("meme.memes", formats)
	checkError(err)
	configs := p.c.GetMap("meme.memeconfigs", map[string]string{})
	configs[values.Name] = values.Config
	err = p.c.SetMap("meme.memeconfigs", configs)
	checkError(err)
}

func (p *MemePlugin) webRoot(w http.ResponseWriter, r *http.Request) {
	var tpl = template.Must(template.New("factoidIndex").Parse(string(memeIndex)))
	tpl.Execute(w, struct{ Nav []bot.EndPoint }{p.bot.GetWebNavigation()})
}

func (p *MemePlugin) img(w http.ResponseWriter, r *http.Request) {
	_, file := path.Split(r.URL.Path)
	id := file
	if img, ok := p.images[id]; ok {
		w.Write(img.repr)
	} else {
		w.WriteHeader(404)
		w.Write([]byte("not found"))
	}
	p.images.cleanup()
}
