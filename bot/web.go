package bot

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func (b *bot) serveRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, rootIndex)
}

func (b *bot) serveNav(w http.ResponseWriter, r *http.Request) {
	enc := json.NewEncoder(w)
	err := enc.Encode(b.GetWebNavigation())
	if err != nil {
		jsonErr, _ := json.Marshal(err)
		w.WriteHeader(500)
		w.Write(jsonErr)
	}
}

// GetWebNavigation returns a list of bootstrap-vue <b-nav-item> links
// The parent <nav> is not included so each page may display it as
// best fits
func (b *bot) GetWebNavigation() []EndPoint {
	endpoints := b.httpEndPoints
	moreEndpoints := b.config.GetArray("bot.links", []string{})
	for _, e := range moreEndpoints {
		link := strings.SplitN(e, ":", 2)
		if len(link) != 2 {
			continue
		}
		endpoints = append(endpoints, EndPoint{link[0], link[1]})
	}
	return endpoints
}

var rootIndex = `
<!DOCTYPE html>
<html lang="en">
<head>
    <!-- Load required Bootstrap and BootstrapVue CSS -->
    <link type="text/css" rel="stylesheet" href="//unpkg.com/bootstrap/dist/css/bootstrap.min.css" />
    <link type="text/css" rel="stylesheet" href="//unpkg.com/bootstrap-vue@latest/dist/bootstrap-vue.min.css" />

    <!-- Load polyfills to support older browsers -->
    <script src="//polyfill.io/v3/polyfill.min.js?features=es2015%2CMutationObserver"></script>

    <!-- Load Vue followed by BootstrapVue -->
    <script src="//unpkg.com/vue@latest/dist/vue.min.js"></script>
    <script src="//unpkg.com/bootstrap-vue@latest/dist/bootstrap-vue.min.js"></script>
    <script src="https://unpkg.com/vue-router"></script>
    <script src="https://unpkg.com/axios/dist/axios.min.js"></script>
    <meta charset="UTF-8">
    <title>Factoids</title>
</head>
<body>

<div id="app">
    <b-navbar>
        <b-navbar-brand>catbase</b-navbar-brand>
        <b-navbar-nav>
            <b-nav-item v-for="item in nav" :href="item.url">{{ item.name }}</b-nav-item>
        </b-navbar-nav>
    </b-navbar>
</div>

<script>
    var app = new Vue({
        el: '#app',
        data: {
            err: '',
            nav: [],
        },
        mounted: function() {
            axios.get('/nav')
                .then(resp => {
                    this.nav = resp.data;
                })
                .catch(err => console.log(err))
        }
    })
</script>
</body>
</html>
`
