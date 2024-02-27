package tappd

import (
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"net/http"
	"os"
	"path"
	"strings"
)

func (p *Tappd) registerWeb() {
	r := chi.NewRouter()
	r.HandleFunc("/", p.serveImage)
	p.b.GetWeb().RegisterWeb(r, "/tappd/{id}")
}

func (p *Tappd) getImg(id string) ([]byte, error) {
	imgData, ok := p.imageMap[id]
	if ok {
		return imgData.Repr, nil
	}
	imgPath := p.c.Get("tappd.imagepath", "tappdimg")
	entries, err := os.ReadDir(imgPath)
	if err != nil {
		log.Error().Err(err).Msgf("can't read image path")
		return nil, err
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), id) {
			return os.ReadFile(path.Join(imgPath, e.Name()))
		}
	}
	log.Error().Msgf("didn't find image")
	return nil, fmt.Errorf("file not found")
}

func (p *Tappd) serveImage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	data, err := p.getImg(id)
	if err != nil {
		log.Error().Err(err).Msgf("error getting image")
		w.WriteHeader(404)
		out, _ := json.Marshal(struct{ err string }{"could not find ID: " + err.Error()})
		w.Write(out)
		return
	}
	w.Write(data)
}
