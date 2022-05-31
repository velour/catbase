package fact

import (
	"embed"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"html/template"
	"net/http"
	"strings"
)

//go:embed *.html
var embeddedFS embed.FS

// Register any web URLs desired
func (p *FactoidPlugin) registerWeb() {
	r := chi.NewRouter()
	r.HandleFunc("/api", p.serveAPI)
	r.HandleFunc("/req", p.serveQuery)
	r.HandleFunc("/", p.serveQuery)
	p.b.RegisterWebName(r, "/factoid", "Factoid")
}

func linkify(text string) template.HTML {
	parts := strings.Fields(text)
	for i, word := range parts {
		if strings.HasPrefix(word, "http") {
			parts[i] = fmt.Sprintf("<a href=\"%s\">%s</a>", word, word)
		}
	}
	return template.HTML(strings.Join(parts, " "))
}

func (p *FactoidPlugin) serveAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fmt.Fprintf(w, "Incorrect HTTP method")
		return
	}
	info := struct {
		Query string `json:"query"`
	}{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&info)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprint(w, err)
		return
	}

	entries, err := getFacts(p.db, info.Query, "")
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprint(w, err)
		return
	}

	data, err := json.Marshal(entries)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprint(w, err)
		return
	}
	w.Write(data)
}

func (p *FactoidPlugin) serveQuery(w http.ResponseWriter, r *http.Request) {
	index, _ := embeddedFS.ReadFile("index.html")
	w.Write(index)
}
