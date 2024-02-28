package fact

import (
	"embed"
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
	r.Post("/search", p.handleSearch)
	r.HandleFunc("/req", p.serveQuery)
	r.HandleFunc("/", p.serveQuery)
	p.b.GetWeb().RegisterWebName(r, "/factoid", "Factoid")
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

func (p *FactoidPlugin) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.FormValue("query")

	entries, err := getFacts(p.db, query, "")
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprint(w, err)
		return
	}

	p.searchResults(entries).Render(r.Context(), w)
}

func (p *FactoidPlugin) serveQuery(w http.ResponseWriter, r *http.Request) {
	p.b.GetWeb().Index("Fact", p.factIndex()).Render(r.Context(), w)
}
