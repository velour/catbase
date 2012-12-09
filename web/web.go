package web

import (
	"html/template"
	"log"
	"net/http"
)

// Define the address which this service will respond
var addr string = "0.0.0.0:6969"

// path to the template files
var tmplPath string = "templates"

// Return an index of the web functions available
func indexHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "index", nil)
}

// Try to load and execute a template for the given site
func renderTemplate(w http.ResponseWriter, r *http.Request, tmpl string, data interface{}) {
	t, err = template.ParseFiles(filepath.Join(tmplPath, tmpl+".html"))
	if err != nil {
		http.Error(w, "Could not load templates.", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	err = t.Execute(w, data)
	if err != nil {
		http.Error(w, "Could not load templates.", http.StatusInternalServerError)
		log.Println(err)
	}
}

func main() {
	http.HandleFunc("/", indexHandler)
	log.Fatal(http.ListenAndServe(addr, nil))
}
