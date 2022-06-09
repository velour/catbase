package emojy

import (
	"embed"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

//go:embed *.html
var embeddedFS embed.FS

func (p *EmojyPlugin) registerWeb() {
	r := chi.NewRouter()
	r.HandleFunc("/all", p.handleAll)
	r.HandleFunc("/allFiles", p.handleAllFiles)
	r.HandleFunc("/upload", p.handleUpload)
	r.HandleFunc("/file/{name}", p.handleEmojy)
	r.HandleFunc("/", p.handleIndex)
	p.b.RegisterWebName(r, "/emojy", "Emojys")
}

func (p *EmojyPlugin) handleIndex(w http.ResponseWriter, r *http.Request) {
	index, _ := embeddedFS.ReadFile("index.html")
	w.Write(index)
}

func (p *EmojyPlugin) handleAll(w http.ResponseWriter, r *http.Request) {
	emojy, err := p.allCounts()
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
	_, urlMap, err := p.allFiles()
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

func (p *EmojyPlugin) handleUpload(w http.ResponseWriter, r *http.Request) {
	enc := json.NewEncoder(w)
	newFilePath, err := p.FileSave(r)
	if err != nil {
		log.Error().Err(err).Msgf("could not upload file")
		w.WriteHeader(500)
		enc.Encode(struct{ err error }{fmt.Errorf("file not saved: %s", err)})
		return
	}
	log.Debug().Msgf("uploaded file to %s", newFilePath)
	w.WriteHeader(200)
	enc.Encode(struct{ file string }{newFilePath})

}

func (p *EmojyPlugin) FileSave(r *http.Request) (string, error) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		if err != http.ErrNotMultipart {
			return "", err
		}
		if err := r.ParseForm(); err != nil {
			return "", err
		}
	}
	if r.MultipartForm == nil || len(r.MultipartForm.File) == 0 {
		return "", fmt.Errorf("no files")
	}

	password := r.Form.Get("password")
	if password != p.b.GetPassword() {
		return "", fmt.Errorf("incorrect password")
	}

	for _, fileHeaders := range r.MultipartForm.File {
		for _, fileHeader := range fileHeaders {
			body, err := fileHeader.Open()
			if err != nil {
				return "", fmt.Errorf("error opening part %q: %s", fileHeader.Filename, err)
			}
			emojyFileName := fileHeader.Filename
			emojyName := strings.TrimSuffix(emojyFileName, filepath.Ext(emojyFileName))
			if ok, _, _, _ := p.isKnownEmojy(emojyName); ok {
				return "", fmt.Errorf("emojy already exists")
			}
			emojyPath := p.c.Get("emojy.path", "emojy")
			contentType := fileHeader.Header.Get("Content-Type")
			if !strings.HasPrefix(contentType, "image") {
				return "", fmt.Errorf("incorrect mime type - given: %s", contentType)
			}
			fullPath := filepath.Clean(filepath.Join(emojyPath, emojyFileName))
			_ = os.MkdirAll(emojyPath, os.ModePerm)
			log.Debug().Msgf("trying to create/open file: %s", fullPath)
			file, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE, os.ModePerm)
			if err != nil {
				return "", err
			}
			defer file.Close()
			_, err = io.Copy(file, body)
			if err != nil {
				return "", err
			}
			return emojyFileName, nil
		}
	}
	return "", fmt.Errorf("did not find file")
}

func (p *EmojyPlugin) handleEmojy(w http.ResponseWriter, r *http.Request) {
	fname := chi.URLParam(r, "name")
	emojyPath := p.c.Get("emojy.path", "emojy")
	contents, err := ioutil.ReadFile(path.Join(emojyPath, fname))
	if err != nil {
		w.WriteHeader(404)
		out, _ := json.Marshal(struct{ err error }{err})
		w.Write(out)
	}
	w.Write(contents)
}
