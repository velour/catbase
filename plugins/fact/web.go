package fact

import (
	"embed"
	"fmt"
	"github.com/ggicci/httpin"
	"github.com/go-chi/chi/v5"
	"net/http"
)

//go:embed *.html
var embeddedFS embed.FS

// Register any web URLs desired
func (p *FactoidPlugin) registerWeb() {
	r := chi.NewRouter()
	r.With(httpin.NewInput(SearchReq{})).
		Post("/search", p.handleSearch)
	r.Get("/", p.serveQuery)
	p.b.GetWeb().RegisterWebName(r, "/factoid", "Factoid")
}

type SearchReq struct {
	Query string `in:"form=query"`
}

func (p *FactoidPlugin) handleSearch(w http.ResponseWriter, r *http.Request) {
	input := r.Context().Value(httpin.Input).(*SearchReq)

	entries, err := getFacts(p.db, input.Query, "")
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
