package emojy

import (
	"embed"
	"encoding/json"
	"fmt"
	"github.com/ggicci/httpin"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

//go:embed *.html
var embeddedFS embed.FS

func (p *EmojyPlugin) registerWeb() {
	r := chi.NewRouter()
	r.HandleFunc("/all", p.handleAll)
	r.HandleFunc("/allFiles", p.handleAllFiles)
	r.With(httpin.NewInput(UploadReq{})).
		Post("/upload", p.handleUpload)
	r.With(httpin.NewInput(EmojyReq{})).
		HandleFunc("/file/{name}", p.handleEmojy)
	r.HandleFunc("/stats", p.handleStats)
	r.HandleFunc("/list", p.handleList)
	r.HandleFunc("/new", p.handleUploadForm)
	r.HandleFunc("/", p.handleStats)
	p.b.GetWeb().RegisterWebName(r, "/emojy", "Emojys")
}

type emojyMap map[string][]EmojyCount

func (p *EmojyPlugin) handleUploadForm(w http.ResponseWriter, r *http.Request) {
	p.b.GetWeb().Index("Emojy", p.uploadIndex()).Render(r.Context(), w)
}

func (p *EmojyPlugin) handleList(w http.ResponseWriter, r *http.Request) {
	threshold := p.c.GetInt("emojy.statthreshold", 1)
	emojy, err := p.allCountsFlat(threshold)
	if err != nil {
		fmt.Fprintf(w, "Error: %s", err)
		return
	}
	p.b.GetWeb().Index("Emojy", p.listTempl(emojy)).Render(r.Context(), w)
}

func (p *EmojyPlugin) handleStats(w http.ResponseWriter, r *http.Request) {
	threshold := p.c.GetInt("emojy.statthreshold", 1)
	emojy, err := p.allCounts(threshold)
	if err != nil {
		w.WriteHeader(500)
		log.Error().Err(err).Msgf("handleAll")
		out, _ := json.Marshal(struct{ err error }{err})
		w.Write(out)
		return
	}
	p.b.GetWeb().Index("Emojy", p.statsIndex(emojy)).Render(r.Context(), w)
}

func (p *EmojyPlugin) handlePage(file string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		index, _ := embeddedFS.ReadFile(file)
		w.Write(index)
	}
}

func (p *EmojyPlugin) handleAll(w http.ResponseWriter, r *http.Request) {
	threshold := p.c.GetInt("emojy.statthreshold", 1)
	emojy, err := p.allCounts(threshold)
	if err != nil {
		w.WriteHeader(500)
		log.Error().Err(err).Msgf("handleAll")
		out, _ := json.Marshal(struct{ err error }{err})
		w.Write(out)
	}
	out, _ := json.Marshal(emojy)
	w.Write(out)
}

func (p *EmojyPlugin) handleAllFiles(w http.ResponseWriter, r *http.Request) {
	_, urlMap, err := AllFiles(p.emojyPath, p.baseURL)
	if err != nil {
		w.WriteHeader(500)
		out, _ := json.Marshal(struct{ err error }{err})
		w.Write(out)
		return
	}
	enc := json.NewEncoder(w)
	err = enc.Encode(urlMap)
	if err != nil {
		w.WriteHeader(500)
		out, _ := json.Marshal(struct{ err error }{err})
		w.Write(out)
	}
}

type UploadReq struct {
	Password   string       `in:"form=password"`
	Attachment *httpin.File `in:"form=attachment"`
}

func (p *EmojyPlugin) handleUpload(w http.ResponseWriter, r *http.Request) {
	input := r.Context().Value(httpin.Input).(*UploadReq)
	log.Printf("handleUpload: %#v", input)
	newFilePath, err := p.FileSave(input)
	if err != nil {
		log.Error().Err(err).Msgf("could not upload file")
		w.WriteHeader(500)
		fmt.Fprintf(w, "Error with file upload")
		return
	}
	log.Debug().Msgf("uploaded file to %s", newFilePath)
	w.WriteHeader(200)
	fmt.Fprintf(w, "success")
}

func (p *EmojyPlugin) FileSave(input *UploadReq) (string, error) {
	if !p.b.CheckPassword("", input.Password) {
		return "", fmt.Errorf("incorrect password")
	}

	file := input.Attachment
	emojyFileName := file.Filename()
	emojyName := strings.TrimSuffix(emojyFileName, filepath.Ext(emojyFileName))
	if ok, _, _, _ := p.isKnownEmojy(emojyName); ok {
		return "", fmt.Errorf("emojy already exists")
	}
	contentType := file.MIMEHeader().Get("Content-Type")
	if !strings.HasPrefix(contentType, "image") {
		return "", fmt.Errorf("incorrect mime type - given: %s", contentType)
	}
	fullPath := filepath.Clean(filepath.Join(p.emojyPath, emojyFileName))
	_ = os.MkdirAll(p.emojyPath, os.ModePerm)
	log.Debug().Msgf("trying to create/open file: %s", fullPath)
	outFile, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE, os.ModePerm)
	if err != nil {
		return "", err
	}
	defer outFile.Close()
	inFile, _ := file.ReadAll()
	stream, _ := file.OpenReceiveStream()
	_, err = outFile.Write(inFile)
	_, err = io.Copy(outFile, stream)
	if err != nil {
		return "", err
	}
	return emojyFileName, nil
}

type EmojyReq struct {
	Name string `in:"path=name"`
}

func (p *EmojyPlugin) handleEmojy(w http.ResponseWriter, r *http.Request) {
	input := r.Context().Value(httpin.Input).(*EmojyReq)
	contents, err := ioutil.ReadFile(path.Join(p.emojyPath, input.Name))
	if err != nil {
		w.WriteHeader(404)
		out, _ := json.Marshal(struct{ err error }{err})
		w.Write(out)
	}
	w.Write(contents)
}
